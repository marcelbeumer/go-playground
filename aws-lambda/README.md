# aws-lambda

Basic AWS lambda example, deployed using terraform.

```bash
make all
terraform init
terraform apply
```

Afterwards you can build, zip and upload to AWS without running terraform with:

```bash
make deploy
```
