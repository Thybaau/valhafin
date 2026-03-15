# Swagger API Documentation

## Accès

Avec le backend lancé :

```
http://localhost:8080/swagger/index.html
```

## Comment ça marche

La documentation est générée automatiquement par [swaggo/swag](https://github.com/swaggo/swag) à partir des commentaires Go dans le code source.

- Les métadonnées générales (titre, version, host) sont dans `main.go`
- Les annotations par endpoint sont dans `internal/api/handlers.go`, au-dessus de chaque handler
- Les fichiers générés (`docs.go`, `swagger.json`, `swagger.yaml`) sont dans `internal/docs/`
- La route `/swagger/` est déclarée dans `internal/api/routes.go`

## Syntaxe des annotations

Exemple d'annotation sur un handler :

```go
// @Summary Lister tous les comptes
// @Description Récupère la liste de tous les comptes financiers
// @Tags accounts
// @Produce json
// @Success 200 {array} models.Account
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts [get]
func (h *Handler) GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
```

Annotations courantes :
- `@Summary` — titre court
- `@Description` — description détaillée
- `@Tags` — catégorie dans l'UI
- `@Accept` / `@Produce` — formats (json, multipart/form-data)
- `@Param` — paramètres (path, query, body, formData)
- `@Success` / `@Failure` — codes HTTP et types de réponse
- `@Router` — route et méthode HTTP

## Mettre à jour la documentation

Après avoir modifié ou ajouté des annotations dans les handlers :

```bash
swag init --output internal/docs --parseDependency --parseInternal
```

Cela régénère les 3 fichiers dans `internal/docs/`. Pensez à les commiter.

## Installation de swag (si besoin)

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```
