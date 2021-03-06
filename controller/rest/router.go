/*
 Copyright © 2020 The OpenEBS Authors

 This file was originally authored by Rancher Labs
 under Apache License 2018.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package rest

import (
	"net/http"
	_ "net/http/pprof" /* for profiling */

	"github.com/gorilla/mux"
	"github.com/openebs/jiva/replica/rest"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rancher/go-rancher/api"
)

func NewRouter(s *Server) *mux.Router {
	schemas := NewSchema()
	router := mux.NewRouter().StrictSlash(true)
	f := rest.HandleError

	// API framework routes
	router.Methods("GET").Path("/").Handler(api.VersionsHandler(schemas, "v1"))
	router.Methods("GET").Path("/v1/schemas").Handler(api.SchemasHandler(schemas))
	router.Methods("GET").Path("/v1/schemas/{id}").Handler(api.SchemaHandler(schemas))
	router.Methods("GET").Path("/v1").Handler(api.VersionHandler(schemas, "v1"))

	// Volumes
	router.Methods("GET").Path("/v1/volumes").Handler(f(schemas, s.ListVolumes))
	router.Methods("GET").Path("/v1/volumes/{id}").Handler(f(schemas, s.GetVolume))
	router.Methods("GET").Path("/v1/stats").Handler(f(schemas, s.GetVolumeStats))
	router.Methods("GET").Path("/v1/checkpoint").Handler(f(schemas, s.GetCheckpoint))
	router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "start").Handler(f(schemas, s.StartVolume))
	router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "shutdown").Handler(f(schemas, s.ShutdownVolume))
	router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "snapshot").Handler(f(schemas, s.SnapshotVolume))
	router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "revert").Handler(f(schemas, s.RevertVolume))
	router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "resize").Handler(f(schemas, s.ResizeVolume))
	router.Methods("POST").Path("/v1/volumes/{id}").Queries("action", "setlogging").Handler(f(schemas, s.SetLogging))
	router.Methods("DELETE").Path("/v1/volumes/{id}").Queries("action", "deleteSnapshot").Handler(f(schemas, s.DeleteSnapshot))
	// Replicas
	router.Methods("GET").Path("/v1/replicas").Handler(f(schemas, s.ListReplicas))
	router.Methods("GET").Path("/v1/replicas/{id}").Handler(f(schemas, s.GetReplica))
	router.Methods("POST").Path("/v1/register").Handler(f(schemas, s.RegisterReplica))
	router.Methods("POST").Path("/v1/replicas").Handler(f(schemas, s.CreateReplica))
	router.Methods("POST").Path("/v1/quorumreplicas").Handler(f(schemas, s.CreateQuorumReplica))
	router.Methods("POST").Path("/v1/replicas/{id}").Queries("action", "preparerebuild").Handler(f(schemas, s.PrepareRebuildReplica))
	router.Methods("POST").Path("/v1/replicas/{id}").Queries("action", "verifyrebuild").Handler(f(schemas, s.VerifyRebuildReplica))
	router.Methods("DELETE").Path("/v1/replicas/{id}").Handler(f(schemas, s.DeleteReplica))
	router.Methods("PUT").Path("/v1/replicas/{id}").Handler(f(schemas, s.UpdateReplica))
	router.Handle("/metrics", promhttp.Handler())

	// Journal
	router.Methods("POST").Path("/v1/journal").Handler(f(schemas, s.ListJournal))
	// Delete
	router.Methods("POST").Path("/v1/delete").Handler(f(schemas, s.DeleteVolume))
	// Debug
	router.Methods("POST").Path("/timeout").Handler(f(schemas, s.AddTimeout))

	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)

	return router
}
