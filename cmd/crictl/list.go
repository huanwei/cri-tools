/*
created by Jiahao@harmonycloud  2019/11/5
*/

package main

import (
	"encoding/json"
	"fmt"
	"github.com/docker/go-units"
	"github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
	"io/ioutil"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"log"
	"path/filepath"
	"strings"
	"time"
)

const (
	STORAGEROOT = "/etc/containers/storage.conf"
)

type hcListMessage struct {
	ContainerId string
	CTM         string
	State       string
	Name        string
	PID         string
	IP          string
	MountPoint  string
}

type hcListResult struct {
	Containers []hcListMessage
}

var hcListCommand = cli.Command{
	Name:  "ls",
	Usage: "List process of containers",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "pid, p",
			Value: "",
			Usage: "Filter by pid",
		},
		cli.StringFlag{
			Name:  "state, s",
			Value: "",
			Usage: "Filter by container state",
		},
		cli.StringFlag{
			Name:  "name, n",
			Value: "",
			Usage: "filter by container name regular expression pattern",
		},
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "Show all containers",
		},
		cli.StringFlag{
			Name:  "output, o",
			Usage: "Output format, One of: json|yaml|table",
		},
		cli.BoolFlag{
			Name:  "no-trunc",
			Usage: "Show output without truncating the ID",
		},
	},
	Action: func(context *cli.Context) error {
		var err error
		if err = getRuntimeClient(context); err != nil {
			return err
		}
		if err = getImageClient(context); err != nil {
			return err
		}
		opts := hcListOptions{
			all:        context.Bool("all"),
			pid:        context.String("pid"),
			state:      context.String("state"),
			nameRegexp: context.String("name"),
			noTrunc:    context.Bool("no-trunc"),
			output:     context.String("output"),
		}

		if err = hcListContainers(runtimeClient, opts); err != nil {
			return fmt.Errorf("listing containers failed: %v", err)
		}

		return nil
	},
}

func hcListContainers(client pb.RuntimeServiceClient, opts hcListOptions) error {
	filter := &pb.ContainerFilter{}
	st := &pb.ContainerStateValue{}
	if !opts.all {
		st.State = pb.ContainerState_CONTAINER_RUNNING
		filter.State = st
	}
	if opts.state != "" {
		st.State = pb.ContainerState_CONTAINER_UNKNOWN
		switch strings.ToLower(opts.state) {
		case "created":
			st.State = pb.ContainerState_CONTAINER_CREATED
			filter.State = st
		case "running":
			st.State = pb.ContainerState_CONTAINER_RUNNING
			filter.State = st
		case "exited":
			st.State = pb.ContainerState_CONTAINER_EXITED
			filter.State = st
		case "unknown":
			st.State = pb.ContainerState_CONTAINER_UNKNOWN
			filter.State = st
		default:
			log.Fatalf("--state should be one of created, running, exited or unknown")
		}
	}

	request := &pb.ListContainersRequest{
		Filter: filter,
	}
	r, err := client.ListContainers(context.Background(), request)
	if err != nil {
		return err
	}

	storageOpts := StoreOptions{}
	ReloadConfigurationFile(STORAGEROOT, &storageOpts)

	root := filepath.Join(storageOpts.GraphRoot, storageOpts.GraphDriverName+"-containers")

	result := hcListResult{}
	for _, c := range r.Containers {
		if !matchesRegex(opts.nameRegexp, c.Metadata.Name) {
			continue
		}
		createdAt := time.Unix(0, c.CreatedAt)
		ctm := units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"
		id := c.Id
		configRoot := filepath.Join(root, id, "userdata", "config.json")
		stateRoot := filepath.Join(root, id, "userdata", "state.json")
		configJson, err := ioutil.ReadFile(configRoot)
		if err != nil {
			return err
		}
		stateJson, err := ioutil.ReadFile(stateRoot)
		if err != nil {
			return err
		}
		mountPoint := gjson.Get(string(configJson), "root.path").String()
		pid := gjson.Get(string(stateJson), "pid").String()
		IP := gjson.Get(string(stateJson), "annotations.io\\.kubernetes\\.cri-o\\.IP").String()
		fmt.Println(opts.noTrunc)
		if !opts.noTrunc {
			id = getTruncatedID(id, "")
		}
		message := hcListMessage{
			ContainerId: id,
			CTM:         ctm,
			State:       convertContainerState(c.State),
			Name:        c.Metadata.Name,
			PID:         pid,
			IP:          IP,
			MountPoint:  mountPoint,
		}

		// filter by pid
		if opts.pid != "" {
			if opts.pid == pid {
				result.Containers = append(result.Containers, message)
				break
			} else {
				continue
			}
		}

		result.Containers = append(result.Containers, message)
	}

	switch opts.output {
	case "json":
		return outputAsJSON(result)
	case "yaml":
		return outputAsYAML(result)
	case "table", "":
		return outputAsTable(result)
	default:
		return fmt.Errorf("unsupported output format %q", opts.output)
	}

	return nil
}

func outputAsJSON(obj hcListResult) error {
	jsonBytes, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}

func outputAsYAML(obj hcListResult) error {
	yamlBytes, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	fmt.Println(string(yamlBytes))
	return nil
}

func outputAsTable(obj hcListResult) error {
	display := newTableDisplay(20, 1, 3, ' ', 0)
	display.AddRow([]string{columnContainer, columnCreated, columnState, columnName, columnPID, columnIP, columnMountPoint})

	for _, r := range obj.Containers {
		display.AddRow([]string{getTruncatedID(r.ContainerId, ""), r.CTM, r.State, r.Name,
			r.PID, r.IP, r.MountPoint})
	}
	_ = display.Flush()
	return nil
}
