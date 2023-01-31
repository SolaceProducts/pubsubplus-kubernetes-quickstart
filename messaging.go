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

package messaging

import (
	"solace.dev/go/messaging/internal/impl"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/message/rgmid"
)

// NewMessagingServiceBuilder returns an instance of solace.MessagingServiceBuilder that can be used
// to build a new solace.MessagingService.
func NewMessagingServiceBuilder() solace.MessagingServiceBuilder {
	return impl.NewMessagingServiceBuilder()
}

// ReplicationGroupMessageIDOf returns an instance of rmid.ReplicationGroupMessageID from the given
// string that can be compared to a message's ReplicationGroupMessageID or used to start replay.
// Replication Group Message IDs come in the form rmid1:xxxxx-xxxxxxxxxxx-xxxxxxxx-xxxxxxxx
func ReplicationGroupMessageIDOf(rmidAsString string) (replicationGroupMessageID rgmid.ReplicationGroupMessageID, err error) {
	return impl.ReplicationGroupMessageIDOf(rmidAsString)
}
