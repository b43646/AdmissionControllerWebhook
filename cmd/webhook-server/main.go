/*
Copyright (c) 2019 StackRox Inc.

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

package main

import (
	"fmt"
	"os"
	"strings"
	"encoding/json"
	//	"errors"
	//	"fmt"
	"k8s.io/api/admission/v1beta1"
	//	corev1 "k8s.io/api/core/v1"
	//ev1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ov1 "github.com/openshift/api/apps/v1"
	"log"
	"net/http"
	"path/filepath"
)

const (
	tlsDir      = `/run/secrets/tls`
	tlsCertFile = `tls.crt`
	tlsKeyFile  = `tls.key`
)

var (
	//podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
	//deploymentResource = metav1.GroupVersionResource{Version: "extensions/v1beta1", Resource: "Deployments"}
	dcResource = metav1.GroupVersionResource{Version: "apps.openshift.io/v1", Resource: "DeploymentConfig"}
)

// applySecurityDefaults implements the logic of our example admission controller webhook. For every pod that is created
// (outside of Kubernetes namespaces), it first checks if `runAsNonRoot` is set. If it is not, it is set to a default
// value of `false`. Furthermore, if `runAsUser` is not set (and `runAsNonRoot` was not initially set), it defaults
// `runAsUser` to a value of 1234.
//
// To demonstrate how requests can be rejected, this webhook further validates that the `runAsNonRoot` setting does
// not conflict with the `runAsUser` setting - i.e., if the former is set to `true`, the latter must not be `0`.
// Note that we combine both the setting of defaults and the check for potential conflicts in one webhook; ideally,
// the latter would be performed in a validating webhook admission controller.
func applySecurityDefaults(req *v1beta1.AdmissionRequest) ([]patchOperation, error) {
	// This handler should only get called on Pod objects as per the MutatingWebhookConfiguration in the YAML file.
	// However, if (for whatever reason) this gets invoked on an object of a different kind, issue a log message but
	// let the object request pass through otherwise.
	//if req.Resource != podResource {
	//	log.Printf("expect resource to be %s", podResource)
	//	return nil, nil
	//}

	// Parse the Pod object.
	//raw := req.Object.Raw
	//pod := corev1.Pod{}
	//if _, _, err := universalDeserializer.Decode(raw, nil, &pod); err != nil {
	//	return nil, fmt.Errorf("could not deserialize pod object: %v", err)
	//}

	var patches []patchOperation

	if req.Resource != dcResource {
		log.Printf("expect resource to be %s", dcResource)
		return nil, nil
	} else {
		raw := req.Object.Raw
		dc := &ov1.DeploymentConfig{}

		//if _, _, err := universalDeserializer.Decode(raw, nil, dc); err != nil {
		//	return nil, fmt.Errorf("could not deserialize pod object: %v", err)
		//}

		if err := json.Unmarshal(raw, dc); err != nil {
			return nil, fmt.Errorf("could not unmarshal dc object: #{err}")
		}
		oldRegistry := "ubuntu"
		newRegistry := "loren"

		for i := 0; i < len(dc.Spec.Template.Spec.Containers); i++ {
			imageAddress := dc.Spec.Template.Spec.Containers[i].Image
			//oldRegistry := os.Getenv("OLD_REGISTRY")
			//newRegistry := os.Getenv("NEW_REGISTRY")

			newImageAddress := strings.Replace(imageAddress, oldRegistry, newRegistry, 1)

			path := fmt.Sprintf("/spec/template/spec/containers/%d/image", i)

			patches = append(patches, patchOperation{
				Op:    "replace",
				Path:  path,
				Value: newImageAddress,
			})
		}
	}

	//// Create patch operations to apply sensible defaults, if those options are not set explicitly.
	//var patches []patchOperation
	//patches = append(patches, patchOperation{
	//	Op:    "replace",
	//	Path:  "/metadata/labels/user",
	//	Value: "Loren",
	//})

	return patches, nil
}

func main() {
	certPath := filepath.Join(tlsDir, tlsCertFile)
	keyPath := filepath.Join(tlsDir, tlsKeyFile)

	mux := http.NewServeMux()
	mux.Handle("/mutate", admitFuncHandler(applySecurityDefaults))
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    ":8443",
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}
