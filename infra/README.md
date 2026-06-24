# infra/

Pulumi (Go SDK) program that provisions the AWS resources for the-run.

## Resources

- Route53 hosted zone for `<domain>`.
- ACM cert in `us-east-1` for the site (CloudFront requires us-east-1).
- ACM cert in `eu-north-1` for the API (regional endpoint).
- S3 bucket + BPA + SSE for the frontend static assets.
- CloudFront distribution with Origin Access Control to the bucket.
- Synced folder uploading `../frontend/build/`.
- Lambda function (`provided.al2023`, arm64, handler `bootstrap`).
- API Gateway v2 HTTP API with `AWS_PROXY` integration to the Lambda.
- Custom API Gateway domain on `api.<domain>`.
- Route53 alias records for the site and API.

## Stack config

| Key                          | Example         | Notes                                                      |
| ---------------------------- | --------------- | ---------------------------------------------------------- |
| `aws:region`                 | `eu-north-1`    | Default provider region.                                   |
| `the-run:domain`             | `example.com`   | Required.                                                  |
| `the-run:apiSubdomain`       | `api`           | Defaults to `api`.                                         |
| `the-run:siteSubdomain`      | `` (empty)      | Empty = apex. Set to `www` (or similar) for a subdomain.   |
| `the-run:createHostedZone`   | `true`          | Set to `false` if the zone already exists in Route53.      |

## Prerequisites before `pulumi up`

1. `AWS_PROFILE` set, `aws sts get-caller-identity` works.
2. `pulumi login s3://...` to the state bucket.
3. `PULUMI_CONFIG_PASSPHRASE` exported.
4. `../backend/dist/lambda.zip` exists (built with the cross-compile command in
   the root README).
5. `../frontend/build/` populated (built with `PUBLIC_API_BASE_URL` set to the
   eventual production URL).

If either build directory is missing, `pulumi up` fails fast.

## Common operations

```sh
pulumi preview          # diff without applying
pulumi up               # apply
pulumi stack output     # show all outputs
pulumi destroy          # tear down (frontend bucket force-destroys)
```

## Cert validation deadlock

On the first `pulumi up` you'll see ACM cert resources sit in `creating…`.
That's because validation requires DNS records to resolve, and the registrar
doesn't yet know about the new Route53 zone. Two options:

- **Run `pulumi up --target` on just the zone first**, copy the
  `hostedZoneNameServers` output to your registrar, then re-run a full
  `pulumi up`. The deploy will pick up where it left off.
- **Or just let it hang**: leave `pulumi up` running while you update the
  registrar. Once DNS propagates, validation completes and the rest of the
  stack proceeds.

CloudFront then takes another 5–15 minutes to roll out globally on first
create. This is normal.
