package dns

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"text/template"

	"github.com/code-ready/crc/pkg/crc/services"
)

const (
	dnsmasqConfTemplate = `user=root
port= {{ .Port }}
bind-interfaces
expand-hosts
log-queries
srv-host=_etcd-server-ssl._tcp.{{ .ClusterName}}.{{ .BaseDomain }},etcd-0.{{ .ClusterName}}.{{ .BaseDomain }},2380,10
local=/{{ .ClusterName}}.{{ .BaseDomain }}/
domain={{ .ClusterName}}.{{ .BaseDomain }}
address=/{{ .AppsDomain }}/{{ .IP }}
address=/etcd-0.{{ .ClusterName}}.{{ .BaseDomain }}/{{ .IP }}
address=/api.{{ .ClusterName}}.{{ .BaseDomain }}/{{ .IP }}
address=/api-int.{{ .ClusterName}}.{{ .BaseDomain }}/{{ .IP }}
addn-hosts=/etc/hosts.openshift
`
	dnsmasqHostsTemplate = `{{.InternalIP}} {{ .Hostname }}.{{ .ClusterName}}.{{ .BaseDomain }}
{{ .IP }} {{ .Hostname }}.{{ .ClusterName}}.{{ .BaseDomain }}
`
)

type dnsmasqConfFileValues struct {
	BaseDomain  string
	Port        int
	ClusterName string
	Hostname    string
	IP          string
	AppsDomain  string
	InternalIP  string
}

func createDnsmasqDNSConfig(serviceConfig services.ServicePostStartConfig) error {
	domain := serviceConfig.BundleMetadata.ClusterInfo.BaseDomain

	dnsmasqConfFileValues := dnsmasqConfFileValues{
		BaseDomain:  domain,
		Hostname:    serviceConfig.BundleMetadata.Nodes[0].Hostname,
		Port:        dnsServicePort,
		AppsDomain:  serviceConfig.BundleMetadata.ClusterInfo.AppsDomain,
		ClusterName: serviceConfig.BundleMetadata.ClusterInfo.ClusterName,
		IP:          serviceConfig.IP,
		InternalIP:  serviceConfig.BundleMetadata.Nodes[0].InternalIP,
	}

	dnsConfig, err := createDnsConfigFile(dnsmasqConfFileValues, dnsmasqConfTemplate)
	if err != nil {
		return err
	}

	encodeddnsConfig := base64.StdEncoding.EncodeToString([]byte(dnsConfig))
	_, err = serviceConfig.SSHRunner.Run(
		fmt.Sprintf("echo '%s' | openssl enc -base64 -d | sudo tee /var/srv/dnsmasq.conf > /dev/null",
			encodeddnsConfig))
	if err != nil {
		return err
	}

	dnsHostConfig, err := createDnsConfigFile(dnsmasqConfFileValues, dnsmasqHostsTemplate)
	if err != nil {
		return err
	}

	encodeddnsConfig = base64.StdEncoding.EncodeToString([]byte(dnsHostConfig))
	_, err = serviceConfig.SSHRunner.Run(
		fmt.Sprintf("echo '%s' | openssl enc -base64 -d | sudo tee /var/srv/hosts.openshift > /dev/null",
			encodeddnsConfig))
	if err != nil {
		return err
	}

	return nil
}

func createDnsConfigFile(values dnsmasqConfFileValues, tmpl string) (string, error) {
	var dnsConfigFile bytes.Buffer

	t, err := template.New("dnsConfigFile").Parse(tmpl)
	if err != nil {
		return "", err
	}
	err = t.Execute(&dnsConfigFile, values)
	if err != nil {
		return "", err
	}
	return dnsConfigFile.String(), nil
}
