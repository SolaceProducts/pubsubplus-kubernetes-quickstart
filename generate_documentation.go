// pubsubplus-go-client
//
// Copyright 2021-2022 Solace Corporation. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
)

const destination = "./docs/"
const siteNameTemplate = "Solace PubSub+ Messaging API for Go v%s"
const siteFooter = "Copyright 2021-2022 Solace Corporation. All rights reserved."

const siteDescription = `The Solace PubSub+ Messaging API for Go allows developers to create client-based messaging applications that
connect and subscribe to in order to publish/consume messages from PubSub+ event brokers.

## Installing the PubSub+ Messaging API for Go
Run the following to install the Messaging API to your project:
<pre>go get solace.dev/go/messaging</pre>

## Using Secure Socket Layer with the Messaging API for Go
To use secure socket layer (SSL) endpoints, OpenSSL 1.1.1 must installed on the systems that run your client applications.
Client applications use SSL endpoints to create secure connections to an event broker (or broker).
For example, on PubSub+ event brokers, you can use SMF TLS/SSL (default port of 55443) and
Web Transport TLS/SSL connectivity (default port 1443) for messaging. For more details, see the
<a href="solace.dev/go/messaging/pkg/solace/index.html">overview</a> in the package solace.

## Example: Publish a Message Application
This PubSub+ Messaging API uses the builder pattern. 
For simplicity, error checking has been removed from the following example that shows you:
- how to configure and build MessageServiceBuilder to connect to an event broker using basic authentication
- publish a message "hello world" to topic my/topic/string
<pre>
import (
	"time"

	"solace.dev/go/messaging"
	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/resource"
)

func main() {
	// NOTE: All error checking has been removed below and is intended for
	//       to show how to use the API. In production systems, we highly
	//       recommmend you put proper error checking and handling in your code.
	
	// Create a new solace.MessagingServiceBuilder
	messagingServiceBuilder := messaging.NewMessagingServiceBuilder()
	// Configure and build a new solace.MessagingService
	// Each function returns a solace.MessagingServiceBuilder for chaining
	messagingService, _ := messagingServiceBuilder.
		FromConfigurationProvider(config.ServicePropertyMap{
			// Configure the host of the event broker to connect to.
			config.TransportLayerPropertyHost: "localhost",
			// Configure the Message VPN on the event broker for which to connect.
			// Most event brokers have a default Message VPN.
			config.ServicePropertyVPNName: "default",
		}).
		WithAuthenticationStrategy(config.BasicUserNamePasswordAuthentication(
			// In this case, we use basic authentication, so is configured
			// on your event broker (is this the client )
			"username",
			"password",
		)).
		Build()
	
	// Connect the solace.MessagingService using the configuation
	// This call connects this application to the event broker
	_ = messagingService.Connect()
	
	// Configure and build a new solace.DirectMessagePublisher to publish 
	// messages to the event broker
	publisher, _ := messagingService.CreateDirectMessagePublisherBuilder().Build()
	
	// Start the publisher to allow for publishing of outbound messages
	_ = publisher.Start()
	
	// Build a new outbound message
	msg, _ := messagingService.MessageBuilder().BuildWithStringPayload("hello world")
	// Publish the message to the topic my/topic/string
	publisher.Publish(msg, resource.TopicOf("my/topic/string"))
	
	// Terminate the publisher with a grace period of 10 seconds allowing buffered messages to be published
	_ = publisher.Terminate(10 * time.Second)
	
	// Disconnect from the event broker
	_ = messagingService.Disconnect()
}
</pre>
`
const pubsubDocsVersonEnv = "PUBSUB_DOCS_VERSION"

func main() {
	var pubsubDocsVersion string
	if len(os.Args) < 2 {
		fmt.Printf("%s not provided, defaulting to 0.0.0\n", pubsubDocsVersonEnv)
		pubsubDocsVersion = "0.0.0"
	} else {
		pubsubDocsVersion = os.Args[1]
	}

	// assume we are being run from the docs-template directory
	os.Mkdir(destination, os.ModePerm)

	args := []string{
		// set the destination
		"-destination=" + destination,
		// site name formatted string with the version inserted
		"-site-name=" + fmt.Sprintf(siteNameTemplate, pubsubDocsVersion),
		// site description displays on the landing page with markdown
		"-site-description=" + siteDescription,
		// footer set to the copyright
		"-site-footer=" + siteFooter,
		// and target the current module
		".",
	}

	cmd := exec.Command("godoc-static", args...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("An error occurred while generating the documentation: %s\n", err)
		os.Exit(1)
	}
}
