package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// Required auth libraries
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var (
	// RootCmd is the root command of the plugin
	RootCmd = &cobra.Command{
		Use:   "kubectl glance",
		Short: "Kubectl glance",
		RunE: func(cmd *cobra.Command, args []string) error {

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			s.Start()

			GlobalSettings.InitClient()
			w := tabwriter.NewWriter(os.Stdout, 3, 4, 1, ' ', tabwriter.AlignRight)

			nodesOK, nodesKO, cpuAllocatable, memAllocatable, err := GlobalSettings.GetNodes()
			fmt.Fprintln(
				w,
				fmt.Sprintf("Nodes\t %s (%s Unschedulable) Allocatable: CPU %s cores, Mem %s bytes",
					color.GreenString(strconv.Itoa(nodesOK)),
					color.RedString(strconv.Itoa(nodesKO)),
					color.GreenString(cpuAllocatable),
					color.GreenString(memAllocatable),
				),
			)

			namespacedResources, err := GlobalSettings.GetNamespacedRessources()
			for resource, value := range namespacedResources {
				fmt.Fprintln(
					w,
					fmt.Sprintf("%s\t %s",
						strings.Title(resource),
						color.GreenString(strconv.Itoa(value)),
					),
				)
			}

			pv, err := GlobalSettings.GetPersistentVolumes()
			fmt.Fprintln(
				w,
				fmt.Sprintf("Persistent Volumes\t %s",
					color.GreenString(strconv.Itoa(pv)),
				),
			)

			s.Stop()
			w.Flush()
			return err
		},
	}
	// GlobalSettings holds common options/utils
	GlobalSettings *globalSettings
)

type globalSettings struct {
	configFlags *genericclioptions.ConfigFlags
	client      *kubernetes.Clientset
	namespace   string
	restConfig  *rest.Config
	chans       *channels
}

type channels struct {
	pods         chan int
	services     chan int
	configs      chan int
	pvcs         chan int
	sas          chan int
	secrets      chan int
	endpoints    chan int
	daemonsets   chan int
	deploys      chan int
	replicasets  chan int
	statefulsets chan int
	jobs         chan int
	done         chan int
}

func init() {
	flags := genericclioptions.NewConfigFlags()
	flags.AddFlags(RootCmd.PersistentFlags())
	GlobalSettings = &globalSettings{
		configFlags: flags,
	}
}
