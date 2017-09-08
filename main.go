package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const configTemplate = `
# generated configurations by chronus
{{range .}}
Host {{.Name}}
   HostName {{.DnsName}}
   IdentityFile ~/.ssh/{{.KeyName}}.pem
{{end}}
`

type configEntry struct {
	Name    string
	DnsName string
	KeyName string
}

func validateName(name string) bool {
	return name != "" && !strings.Contains(name, " ")
}

func getEntries(region string) ([]configEntry, error) {
	var sess *session.Session
	if region != "" {
		sess = session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		}))
	} else {
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
	}
	ec2cli := ec2.New(sess)

	result, err := ec2cli.DescribeInstances(nil)
	var entries []configEntry
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return nil, err
	}
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			var name string
			for _, tag := range instance.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
					break
				}
			}

			if *instance.KeyName != "" && *instance.PublicDnsName != "" && validateName(name) {
				entries = append(entries, configEntry{
					Name:    name,
					KeyName: *instance.KeyName,
					DnsName: *instance.PublicDnsName,
				})
			}
		}
	}
	return entries, nil
}

var (
	region     = flag.String("region", "", "AWS region name")
	outputPath = flag.String("output", "", "output configuration path, should not ~/.ssh/config (default: stdout)")
)

func init() {
	originalUsage := flag.Usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Reach for the heavens! Engrave the chronicle! It's time to go beyond!\n")
		originalUsage()
	}
	flag.Parse()
}

func openOrStdout(path string) (io.WriteCloser, error) {
	if path == "" {
		return os.Stdout, nil
	}
	return os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
}

func _main() int {
	entries, err := getEntries(*region)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	t := template.New("ssh_config")
	template.Must(t.Parse(configTemplate))

	fp, err := openOrStdout(*outputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer fp.Close()
	t.Execute(fp, entries)

	return 0
}

func main() {
	os.Exit(_main())
}
