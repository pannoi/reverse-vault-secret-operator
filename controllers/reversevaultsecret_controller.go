/*
Copyright 2023.

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

package controllers

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	er "errors"

	vault "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	reversevaultsecretv1beta1 "reverse-vault-secret-operator/api/v1beta1"
)

// ReverseVaultSecretReconciler reconciles a ReverseVaultSecret object
type ReverseVaultSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var contxt = context.Background()
var vaultAddr = os.Getenv("VAULT_HOST")
var vaultToken = os.Getenv("VAULT_TOKEN")
var httpClient = &http.Client{Timeout: 10 * time.Second}

func writeVaultSecret(secretPath string, secretData map[string]interface{}) error {
	hvac, err := vault.NewClient(&vault.Config{Address: vaultAddr, HttpClient: httpClient})
	if err != nil {
		return er.New("Failed to init connection to Vault: " + vaultAddr)
	}
	hvac.SetToken(vaultToken)

	secretPathSlice := strings.SplitN(secretPath, "/", 2)

	_, err = hvac.KVv2(secretPathSlice[0]).Put(contxt, secretPathSlice[1], secretData)
	if err != nil {
		return err
	}

	return nil
}

func readVaultSecret(secretPath string) (map[string]interface{}, error) {
	hvac, err := vault.NewClient(&vault.Config{Address: vaultAddr, HttpClient: httpClient})
	if err != nil {
		return nil, er.New("Failed to init connection to Vault: " + vaultAddr)
	}
	hvac.SetToken(vaultToken)

	secretPathSlice := strings.SplitN(secretPath, "/", 2)

	secret, err := hvac.KVv2(secretPathSlice[0]).Get(contxt, secretPathSlice[1])
	if err != nil {
		return nil, err
	}

	return secret.Data, nil
}

func readKubeSecret(kubeSecret map[string][]byte) (map[string]interface{}, error) {
	resp := make(map[string]interface{})

	for key, val := range kubeSecret {
		resp[key] = string(val)
	}

	return resp, nil
}

func (r *ReverseVaultSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	crData := &reversevaultsecretv1beta1.ReverseVaultSecret{}
	err := r.Get(ctx, req.NamespacedName, crData)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Schema resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ReverseVaultSecret resource")
		return ctrl.Result{}, err
	}

	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: crData.Spec.SecretName, Namespace: crData.Namespace}, secret)
	if err != nil {
		log.Error(err, "Failed to find Secret: "+crData.Spec.SecretName)
		return ctrl.Result{}, err
	}

	vaultSecret, err := readVaultSecret(crData.Spec.VaultPath)
	if err != nil {
		log.Error(err, "Failed to read secret from vault: "+crData.Spec.VaultPath)
		return ctrl.Result{}, err
	}

	kubeSecret, err := readKubeSecret(secret.Data)
	if err != nil {
		log.Error(err, "Failed to convert kube secret to readable format "+secret.Name)
		return ctrl.Result{}, err
	}

	if reflect.DeepEqual(vaultSecret, kubeSecret) {
		return ctrl.Result{Requeue: true}, nil
	}

	err = writeVaultSecret(crData.Spec.VaultPath, kubeSecret)
	if err != nil {
		log.Error(err, "Failed to update secret in vault "+crData.Spec.VaultPath)
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReverseVaultSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&reversevaultsecretv1beta1.ReverseVaultSecret{}).
		Complete(r)
}
