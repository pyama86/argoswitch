package main

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"

	argocdclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	applicationpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/gorilla/handlers"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

var (
	//go:embed templates/index.html
	indexTemplate string
)
var (
	version string
)

const annotationPrimary = "argoswitch.github.io/primary"
const annotationSecondry = "argoswitch.github.io/secondry"
const annotationServiceOut = "argoswitch.github.io/service-out"
const annotationMaint = "argoswitch.github.io/maint"

var annotations = map[string]string{
	"primary":     annotationPrimary,
	"secondry":    annotationSecondry,
	"maint":       annotationMaint,
	"service-out": annotationServiceOut,
}

func changeState(appIf applicationpkg.ApplicationServiceClient, changeTo string, apps []v1alpha1.Application, ctx context.Context) []operation {
	rs := []operation{}
	var err error
	for _, app := range apps {
		for k, v := range app.ObjectMeta.Annotations {
			if k == annotations[changeTo] {
				switch v {
				case "sync":
					_, err = appIf.Sync(ctx, &applicationpkg.ApplicationSyncRequest{
						Name:  &app.ObjectMeta.Name,
						Prune: true,
					})
				case "disable":
					s := app.Spec
					s.SyncPolicy.Automated = nil
					_, err = appIf.UpdateSpec(ctx, &applicationpkg.ApplicationUpdateSpecRequest{
						Name: &app.ObjectMeta.Name,
						Spec: s,
					})

				case "delete":
					_, err = appIf.Delete(ctx, &applicationpkg.ApplicationDeleteRequest{
						Name:    &app.ObjectMeta.Name,
						Cascade: &[]bool{true}[0],
					})

				}

				if err != nil {
					logrus.Error(err)
				}
				rs = append(rs, operation{
					Name:      app.ObjectMeta.Name,
					Operation: v,
					Error:     err,
				})
			}
		}
	}
	return rs
}

var conf Config

type Config struct {
	Listen      string `default:"127.0.0.1:8080"`
	ServerName  string `required:"true"`
	ServerToken string `required:"true"`
	Insecure    bool
	PlainText   bool
}

func stateFilePath() string {
	u, err := user.Current()
	if err != nil {
		return "/tmp/state"
	}
	return path.Join(u.HomeDir, "state")
}
func currentState() string {

	data, err := ioutil.ReadFile(stateFilePath())
	if err != nil {
		return "unknown"
	}
	return string(data)
}

func setState(state string) error {
	return ioutil.WriteFile(stateFilePath(), []byte(state), 0644)
}

func main() {
	err := envconfig.Process("argosw", &conf)
	if err != nil {
		log.Fatal(err)
	}

	r := http.NewServeMux()

	r.HandleFunc("/favicon.ico", http.NotFound)
	r.HandleFunc("/", handleIndex)
	r.HandleFunc("/healthz", handleHealth)

	logrus.Infof("Server listening on %s version %s", conf.Listen, version)
	logrus.Info(http.ListenAndServe(conf.Listen, handlers.LoggingHandler(os.Stdout, r)))
}

type operation struct {
	Name      string
	Operation string
	Error     error
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	clientOpts := &argocdclient.ClientOptions{
		ConfigPath: "",
		ServerAddr: conf.ServerName,
		AuthToken:  conf.ServerToken,
		GRPCWeb:    true,
		Insecure:   conf.Insecure,
		PlainText:  conf.PlainText,
	}

	conn, appIf := argocdclient.NewClientOrDie(clientOpts).NewApplicationClientOrDie()
	defer argoio.Close(conn)
	ctx := context.Background()
	list, err := appIf.List(ctx, &applicationpkg.ApplicationQuery{})
	if err != nil {
		errorResponce(w, err)
		return
	}
	cs := currentState()
	var rs []operation
	if r.Method == "POST" && cs != r.FormValue("action") {
		rs = changeState(appIf, r.FormValue("action"), list.Items, ctx)
		if err := setState(r.FormValue("action")); err != nil {
			errorResponce(w, err)
			return
		}
		cs = r.FormValue("action")
	}

	if err := render(w, detectOperations(list.Items), rs, cs); err != nil {
		errorResponce(w, err)
		return
	}
}

func errorResponce(w http.ResponseWriter, err error) {
	logrus.Error(err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(fmt.Sprintf("500 - %s", err)))
}

func render(w io.Writer, affects map[string][]operation, rs []operation, currentState string) error {
	funcMap := map[string]interface{}{
		"addIcon": func(o operation) string {
			var icon, color string
			switch o.Operation {
			case "sync":
				icon = "check-square"
				color = "cornflowerblue"
			case "disable":
				icon = "stop-circle"
				color = "green"
			case "delete":
				icon = "trash"
				color = "red"
			}
			return fmt.Sprintf(`<i class="bi-%s" style="color: %s;"></i>`, icon, color)
		},
		"safehtml": func(text string) template.HTML {
			return template.HTML(text)
		},
		"errorstr": func(err error) string {
			return err.Error()
		},
	}

	t, err := template.New("index").Funcs(funcMap).Parse(indexTemplate)
	if err != nil {
		return err
	}
	if err := t.Execute(w, struct {
		Annotations  map[string]string
		Affects      map[string][]operation
		Results      []operation
		CurrentState string
	}{
		Annotations:  annotations,
		Affects:      affects,
		Results:      rs,
		CurrentState: currentState,
	}); err != nil {
		return err
	}

	return nil
}

func detectOperations(apps []v1alpha1.Application) map[string][]operation {
	operations := map[string][]operation{}
	for _, app := range apps {
		for k, v := range app.ObjectMeta.Annotations {
			for ak, av := range annotations {
				if k == av {
					operations[ak] = append(operations[ak], operation{
						Name:      app.ObjectMeta.Name,
						Operation: v,
					})
				}
			}
		}
	}
	return operations
}
