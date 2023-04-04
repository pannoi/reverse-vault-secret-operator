# Reverse Vault Secret Operator

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/reverse-vault-secret-operator)](https://artifacthub.io/packages/search?repo=reverse-vault-secret-operator)

The Reverse Vault Secret Operator is going to create monitor kubernetes secret state via `CRD` and if it going to be changed then it will update provided Vault kv/path

> Use case: Might be needed if some kubernetes operator or any automation is interacting with kubernetes secrets directly and you need to re-populate it into [Hashicorp Vault](https://github.com/hashicorp/vault) for different environment, namespaces or whatever from Hashicorp Vault

## Installation

```bash
helm repo add reverse-vault-secret-operator https://pannoi.github.io/reverse-vault-secret-operator-helm/
helm repo update
helm upgrade --install reverse-vault-secret-operator reverse-vault-secret-operator/reverse-vault-secret-operator --values values.yaml
```

You need to set `VAULT_HOST` and `VAULT_TOKEN` initial functionality

```yaml
vault:
  host:
  token:
```

### Generate Vault token
```bash
vault token create -policy=reverse-secret-operator-policy -period=9000h
```

> reverse-secret-operator-policy should have permissions to **READ** and **WRITE** in needed secret engine

## How to use

```yaml
apiVersion: reverse-vault-secret-operator.pannoi/v1beta1
kind: ReverseVaultSecret
metadata:
  name: reverse-vault-secret
spec:
  secretName: my-secret
  vaultPath: kv/path
```
