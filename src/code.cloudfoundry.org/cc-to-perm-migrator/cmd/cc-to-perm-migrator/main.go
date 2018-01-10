package main

import (
	"crypto/tls"
	"os"
	"time"

	"context"

	"net/http"

	"crypto/x509"

	"errors"

	"bytes"

	"sync"

	"code.cloudfoundry.org/cc-to-perm-migrator/cloudcontroller"
	"code.cloudfoundry.org/cc-to-perm-migrator/cmd"
	"code.cloudfoundry.org/cc-to-perm-migrator/httpx"
	"code.cloudfoundry.org/lager"
	flags "github.com/jessevdk/go-flags"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type options struct {
	ConfigFilePath cmd.FileOrStringFlag `long:"config-file-path" description:"Path to the config file for the CloudController migrator" required:"true"`
}

const CloudControllerTimeout = 5 * time.Second

func main() {
	parserOpts := &options{}
	parser := flags.NewParser(parserOpts, flags.Default)
	parser.NamespaceDelimiter = "-"

	_, err := parser.Parse()
	if err != nil {
		// Necessary to not panic because this is how the parser prints Help messages
		os.Exit(1)
	}

	configFileContents, err := parserOpts.ConfigFilePath.Bytes(cmd.OS, cmd.IOReader)
	if err != nil {
		panic(err)
	}

	config, err := cmd.NewConfig(bytes.NewReader(configFileContents))
	if err != nil {
		panic(err)
	}

	logger, _ := config.Logger.Logger("cc-to-perm-migrator")

	uaaCACert, err := config.UAA.CACertPath.Bytes(cmd.OS, cmd.IOReader)
	if err != nil {
		logger.Error("failed-to-read-uaa-ca-cert", err)
		panic(err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(uaaCACert)
	if !ok {
		logger.Error("failed-to-append-certs-from-pem", errors.New("could not append certs"), lager.Data{
			"path": config.UAA.CACertPath,
		})
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	sslcli := &http.Client{Transport: tr}

	tokenURL, err := httpx.JoinURL(logger, config.UAA.URL, "/oauth/token")
	if err != nil {
		panic(err)
	}

	uaaConfig := &clientcredentials.Config{
		ClientID:     config.CloudController.ClientID,
		ClientSecret: config.CloudController.ClientSecret,
		TokenURL:     tokenURL.String(),
		Scopes:       config.CloudController.ClientScopes,
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, sslcli)

	client := uaaConfig.Client(ctx)

	ccClient := cloudcontroller.NewAPIClient(config.CloudController.URL, client, CloudControllerTimeout)

	roleAssignments := make(chan cmd.RoleAssignment)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		cmd.GenerateReport(os.Stdout, roleAssignments)
	}()

	go func() {
		defer wg.Done()
		err = cmd.IterateOverCloudControllerEntities(ctx, logger, roleAssignments, ccClient)

		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()
}
