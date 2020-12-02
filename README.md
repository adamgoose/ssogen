# ssogen

ssogen is a simple utility that produces a valid `~/.aws/config` file populated
with profiles for all available accounts and roles available to the
authenticated AWS SSO user. Just provie your SSO Start URL, no valid CLI session
necessary!

```bash
docker run --rm adamgoose/ssogen https://{myorg}.awsapps.com/start | tee ~/.aws/config.ssogen
```

You'll be prompted to open navigate to a URL, authenticate with AWS SSO, and
click the "Login to AWS CLI" button. Your config file will then be printed to
stdout.

For a full list of options, just ask for help.

```bash
docker run --rm adamgoose/ssogen --help
```