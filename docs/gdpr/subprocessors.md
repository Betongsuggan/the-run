# Underbiträden

> **Auto-generated** from `backend/internal/gdpr/subprocessors.go`.
> Edit that file and run `just gen-gdpr` to refresh.

Ingmarsöloppet anlitar följande underbiträden för att driva tjänsten. Inga personuppgifter delas med någon utöver dessa.

## Amazon Web Services EMEA SARL

- **Roll:** Drift, lagring och e-postutskick
- **Region:** eu-north-1 (Stockholm) + CloudFront-edge inom EES
- **Tjänster som används:** DynamoDB, Lambda, API Gateway, CloudFront, SES, Secrets Manager, CloudWatch Logs
- **Personuppgiftsbiträdesavtal (DPA):** Accepterad via AWS Artifact

## Överföringar till tredje land

Inga. All databehandling sker inom EES — primär region är AWS eu-north-1 (Stockholm); statiska sidor distribueras via CloudFront-edge inom EES.

## Granskningsrytm

Den här listan granskas:

- Vid varje ny extern tjänst som integreras (PR-granskning).
- Årligen som del av den allmänna policygranskningen.
- Före publicering av en ny version av sekretesspolicyn (`/admin/policies` → ny version), så att policytexten matchar listan.
