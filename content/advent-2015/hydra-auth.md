+++
author = ["arekkas"]
date = "2015-12-17T16:05:49+01:00"
linktitle = "Hydra: Run your own IAM service in <5 Minutes"
series = ["Advent 2015"]
title = "Hydra: Run your own Identity and Access Management service in <5 Minutes"
+++

Let me introduce you to [Hydra](https://github.com/ory-am/hydra),
the open source alternative to proprietary authorization solutions in the age of micro services.
It will take you less than five minutes to start up your very own OAuth2 provider and gain access to a rich features set,
including access control and identity management.

[![Hydra](/postimages/advent-2015/hydra.png)](https://github.com/ory-am/hydra)

Hydra was primarily written because we at Ory needed a scalable 12factor OAuth2 consumer / provider with enterprise grade
authorization and interoperability without a ton of dependencies or crazy features. While we where at it, we added
policy, account and client management and other cool features.
Because we can't stand maintaining 5 different databases (or paying someone to maintain them)
and dealing with unpredictable dependency trees, Hydra only requires Go and PostgreSQL (or any SQL speaking database).

Hydra's core features in a nutshell:

* **Account Management**: Sign up, settings, password recovery
* **Access Control / Policy Decision Point / Policy Storage Point** backed by [Ladon](https://github.com/ory-am/ladon).
* Rich set of **OAuth2** features:
  * Hydra implements OAuth2 as specified at [rfc6749](http://tools.ietf.org/html/rfc6749) and [draft-ietf-oauth-v2-10](http://tools.ietf.org/html/draft-ietf-oauth-v2-10) using [osin](https://github.com/RangelReale/osin) and [osin-storage](https://github.com/ory-am/osin-storage)
  * Hydra uses self-contained Acccess Tokens as suggessted in [rfc6794#section-1.4](http://tools.ietf.org/html/rfc6749#section-1.4) by issuing JSON Web Tokens as specified at
   [https://tools.ietf.org/html/rfc7519](https://tools.ietf.org/html/rfc7519) with [RSASSA-PKCS1-v1_5 SHA-256](https://tools.ietf.org/html/rfc7519#section-8) hashing algorithm.
  * Hydra implements **OAuth2 Introspection** ([rfc7662](https://tools.ietf.org/html/rfc7662)) and **OAuth2 Revokation** ([rfc7009](https://tools.ietf.org/html/rfc7009)).
  * Hydra is able to sign users up and in through OAuth2 providers like Dropbox, LinkedIn, Google, you name it.
* Hydra speaks **no HTML**. We believe that the design decision to keep templates out of Hydra is a core feature. *Hydra is backend, not frontend.*
* **Easy command line tools** like `hydra-host jwt` for generating jwt signing key pairs or `hydra-host client create`.
* Hydra works both with **HTTP/2 and TLS** and HTTP (insecure - use only in development).
* Hydra provides many **unit and integration tests**, making sure that everything is as secure as it gets!
We use [github.com/ory-am/dockertest](https://github.com/ory-am/dockertest) for spinning up a postgres (or any other) image on the fly and running integration tests against them.
Give it a try if you want to speed up your integration test development.

Hydra was written by me ([GitHub](https://github.com/arekkas) / [LinkedIn](https://de.linkedin.com/in/aeneasr)) as part of a business application which has not been revealed yet.
Hydra development is supervised by [Thomas Aidan Curran](https://www.linkedin.com/in/thomasaidancurran).

## What do you mean by *Hydra is backend*?

Hydra does not offer a sign in, sign up or authorize HTML page. Instead, if such action is required, Hydra redirects the user
to a predefined URL, for example `http://sign-up-app.yourservice.com/sign-up` or `http://sign-in-app.yourservice.com/sign-in`.
Additionally, a user can authenticate through another OAuth2 Provider, for example Dropbox or Google.

## I want some action, man!

Cool, me too! :) Let me show you how to set up Hydra and get a token for a client app,
also known as the [OAuth2 Client Grant](https://aaronparecki.com/articles/2012/07/29/1/oauth2-simplified#others) (read section "Application Access"). Most of you should be familiar with the console commands. If you're not, feel free to ask if you run into issues in our [GitHub Issue Tracker](https://github.com/ory-am/hydra/issues).

*Please note: Hydra is going to be shipped through a Docker container in the future. For now, you'll need
[Vagrant](https://www.vagrantup.com/), [VirtualBox](https://www.virtualbox.org/) and [Git](https://git-scm.com/).*

```
git clone https://github.com/ory-am/hydra.git
cd hydra
vagrant up
# Get a coffee or wait until the Docker container is released ;)
```

You should now have a running Hydra instance! Vagrant exposes ports 9000 (HTTPS - Hydra) and 9001 (Postgres) on your localhost.
Open [https://localhost:9000/](https://localhost:9000/) to confirm that Hydra is running. You will probably have to add an exception for the
HTTP certificate because it is self-signed, but after that you should see a 404 error indicating that Hydra is running!

Vagrant sets up a test client app (id: app, secret: secret) with super user rights. To do so,
Vagrant runs `hydra-host client create -i app -s secret -r http://localhost:3000/authenticate/callback --as-superuser`.

*hydra-host* offers different capabilities for managing your Hydra instance. Check the [docs](https://github.com/ory-am/hydra#cli-usage) if you want to find out more.
You can also always access hydra-host through vagrant.

```
# Assuming, that your current working directory is /where/you/cloned/hydra
vagrant ssh
hydra-host help
```

*Note: Vagrant sometimes fails to boot due to network issues, if you don't see the 404 error simply run `vagrant destroy -f && vagrant up`. This will take a minute or two,
but Hydra should be running fine after that.*

## OAuth2 Token Client grant

Now that Hydra is running, let's try out some token magic! I'm assuming that you have curl installed on your system. If not, check [this page](http://curl.haxx.se/download.html).
Our goal is to exchange our client credentials for an access token.

```
# --insecure skips SSL verification, do not use this in production.
curl --insecure -X POST --user app:secret "https://localhost:9000/oauth2/token?grant_type=client_credentials"
# You should see something like
{
    "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNTc4MDksImlhdCI6MTQ1MDM1NDIwOSwiaXNzIjoiIiwiamlkIjoiMmYyYjk2MGQtZjE3OC00MzE5LWI4OGEtODc3YzM1Y2U5NTFkIiwibmJmIjoxNDUwMzU0MjA5LCJzdWIiOiJhcHAifQ.cLSY3G0Ngz62hJmanADZ3LUfblB5nOZOWr7bAflE9T0pZBp-Qv1sTkwRCQfqv870cpHdFvN9xL_AReMmNo_o9sLmXfNZDL5WJzDhhsLximxPMD-rO0DjnvY5663l0fvhFMlaGREsHGWDzPN-wZLczRjlFr1JXPv80qMeCm9d343hGMu26WWZ8bfdgAbae8ecmSO_oP7I8U0tWn22FzVJjSRuaShKxlWyQY2K_0-VoHDQDZMTEIXxYGNPA0MmCOEK1DDAiUeKTbguMSLMCjXTkbxd2rMwHday1oHDH8aBkyL0CGmmfVfl20hfRYqJ0x7_0sTd__-inASEjozSvYkVOw",
    "expires_in": 3600,
    "token_type": "Bearer"
}
```

*Note: It is currently under discussion, if Hydra should issue only self containing JWT tokens or support other token types as well.
Feel free to join the discussion our [GitHub Issue Tracker](https://github.com/ory-am/hydra/issues/22).*

## OAuth2 Token Password grant

That was quick, right? Let's try this with a user account!
*Please write the ID you are given done. We will need it later! You should also take note, that this user account does not have super user rights.**

```
# Assuming, that your current working directory is /where/you/cloned/hydra
vagrant ssh
hydra-host account create foo@bar.com --password secret
#
# You'll see something like this. Please write this ID down, we'll need it later!!
#
Created account as "e152f029-424f-4d4d-9d69-643225113ee5".
```

Let's authenticate the user account through the [OAuth2 Password Credentials Workflow](https://aaronparecki.com/articles/2012/07/29/1/oauth2-simplified#others)!

```
curl --insecure --data "grant_type=password&username=foo@bar.com&password=secret" --user app:secret "https://localhost:9000/oauth2/token"
# You should see something like
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNzAwODAsImlhdCI6MTQ1MDM2NjQ4MCwiaXNzIjoiIiwiamlkIjoiYTdjZDFmYWQtZTg5MS00ZDJmLWIwZmEtMzE2Zjg3MTI5ZGIyIiwibmJmIjoxNDUwMzY2NDgwLCJzdWIiOiJiY2U5M2QzMy05YjVhLTQ5MzMtOTQ3Mi1jYWRhMDE4ZGFmNjAifQ.dqUHiAJ0uoUYtV4hqhgVqYqA6PSy1cmNZQruyTpmRaCBh2RHzkijFj4F-T8xTbrFBnysTQG3LxxeXkDNq6PZBsZ4WzvUXSy1R18MayT5FWkgAi-ROQ2lHn9Isw1IgN3XWO-YOaQt9rO0gG4w_hRQ-DprMMKcUkNVC1zK_pdUpaB7cEurYF3sd7krPQjIhucPVhJqDjkAIZGG54kd28_uLqKi3eTaDrViwGLbYzmLenfTb79Hxjfd8qFd_KBQW-f1maLy0BwQNP1pVu2I_P7CBjIwEm898wTPye42CFUfVzyvB6ob4sAZM60YVwzxN_zaw_SO1160HbDI4oO-HwwPig",
  "expires_in": 3600,
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjBlNmJmNTBlLTU1N2EtNGJiYy1iZDk1LTg3ZDJkOTNkOGQ5YSJ9.JFVgu7Tf1BZJLrMbgKi0wyBKXZuHB63yKbv6_UP8TUkUgH8e9S5Gi9MhlPOnU0KyiEkh8p5Z0CMN2HQeIeYj-0p3POFxoSkY6NPZeWKsnPXzDjlJJmXWYrqgI-N-BD26MmoGXLjHt_DY3hxBX_EzHHuqVk9q-2pUAfwc0BHjSidF5EZ852I5e3J0WHbiw4KnogNRKNN-lsiIIEBSjkBxyyH85Dx4JdQZsAJVBKiXXzizWIQeQABAIutvIs5ok3T4xD8WYEiSuiHdKbPKe9bjNGX2OqW1X-eDts4RE0eHWatNQ-IafwMvi-7A0f5PSf26pSGPQ5TyvpA5qbnYAIXrMw",
  "token_type": "Bearer"
}
```

## OAuth2 Authorize Workflow

Ok, let's try the OAuth2 Authorize workflow!
To do this, you'll need the account ID from above. Because Hydra is only backend, I have written
some exemplary sign up and sign in endpoints.
Take a look at them [hydra-signin](https://github.com/ory-am/hydra/blob/master/cli/hydra-signup/main.go)
and [hydra-signup](https://github.com/ory-am/hydra/blob/master/cli/hydra-signup/main.go) to see some very basic example code.

```
# Assuming, that your current working directory is /where/you/cloned/hydra
vagrant ssh
#
# Use the account ID from above
# ACCOUNT_ID=<account_id_from_above> hydra-signin
# for example:
ACCOUNT_ID=e152f029-424f-4d4d-9d69-643225113ee5 hydra-signin &
```

Now, point your web browser to [https://localhost:9000/oauth2/auth?response_type=code&client_id=app&redirect_uri=http://localhost:3000/authenticate/callback&state=foo](https://localhost:9000/oauth2/auth?response_type=code&client_id=app&redirect_uri=http://localhost:3000/authenticate/callback&state=foo).

![Sign in page](/postimages/advent-2015/sign-in.png)

Click where it says "Press this link to sign in". You should now be redirected to another page.

![Sign in callback page](/postimages/advent-2015/sign-in-cb.png)

This location of the sign up and sign in locations are defined the env variables `SIGNUP_URL` and `SIGNUP_URL`.
Obviously, these endpoints are just mock ups. Find more details on environment variables in the [docs](https://github.com/ory-am/hydra/blob/master/README.md#available-environment-variables).

For the next step, you'll need the code you were given. Let's call curl again:

```
# curl --insecure --data "grant_type=authorization_code&code=<code_crom_above>" --user app:secret "https://localhost:9000/oauth2/token"
curl --insecure --data "grant_type=authorization_code&code=fEat4PS3TVeyWrwKgLxICg" --user app:secret "https://localhost:9000/oauth2/token"
# You should see something like
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNTkwNDMsImlhdCI6MTQ1MDM1NTQ0MywiaXNzIjoiIiwiamlkIjoiZmMzODg4NjktZWE2MC00YzE4LWI1NmMtM2I4YmYzOTJmMzU5IiwibmJmIjoxNDUwMzU1NDQzLCJzdWIiOiJhcHAifQ.foEvIJX3hwuCJCQvIi6x31m3g1VQ0RAp6ouiiVFIs2mVM7GsD2O3aS8WxlKaxZ5P7VhbJpxTR2zg9GDSGRe-Acj26r1OVjY9QSoLIeMNg2VfA6AwpASmYhP8EOdlbyjFEK8hC14JXToWn-cT6UXE0IZxg0ANevzDSHlPnaLDemNBkxoQ1cQPIOxPOz7xZSSDZmw9rv-MNlPi6F-FNZOEig5iEyl5vzDgExr5438Qkmc5OzlLYz-RoOroFtiyoqPXp0aYEms4zaowzB4m_DrQd0cIuAKjrtlUnbvId0rOnx-PBtF6yWZfSC7_hmWwtfrmho-XFWfaawjZswRWTAgaMg",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

Cool, you just accomplished the authorize workflow! Let's move on to the next topic, policies!

## Policies

Policies are something very powerful. I have to admit that I am a huge fan of how AWS handles policies and adopted their architecture for Hydra. Please find a more in depth documentation
at the [Ladon GitHub Repository](https://github.com/ory-am/ladon).

```
{
    // This should be a unique ID. This ID is required for database retrieval.
    id: "68819e5a-738b-41ec-b03c-b58a1b19d043",

    // A human readable description. Not required
    description: "something humanly readable",

    // Which identity does this policy affect?
    // As you can see here, you can use regular expressions inside < >.
    subjects: ["max", "peter", "<zac|ken>"],

    // Should the policy allow or deny access?
    effect: "allow",

    // Which resources this policy affects.
    // Again, you can put regular expressions in inside < >.
    resources: ["urn:something:resource_a", "urn:something:resource_b", "urn:something:foo:<.+>"],

    // Which permissions this policy affects. Supports RegExp
    // Again, you can put regular expressions in inside < >.
    permissions: ["<create|delete>", "get"],

    // Under which conditions this policy is active.
    conditions: [
        // Currently, only an exemplary SubjectIsOwner condition is available.
        {
            "op": "SubjectIsOwner"
        }
    ]
}
```

This is what a policy looks like. As you can see, we have various attributes:

* A **Subject** could be an account or an client app
* A **Resource** could be an online article or a file in a cloud drive
* A **Permission** can also be referred to as "Action" ("create" something, "delete" something, ...)
* A **Condition** can be an intelligent assertion *(e.g. is the Subject requesting access also the Resource Owner?)*. Right now, only the SubjectIsOwner Condition is defined. In the future, many more (e.g. IPAddressMatches or UserAgentMatches) will be added.
* The **Effect**, which can only be **allow** or **deny** (deny *always* overrides).

Do you remember that the test client app (*app*) was created with super-user rights and our test user without super-user rights? Let's
see what this means in real life!

Let's see if our test client app has the rights to do "create" on resource "fileA.png". First, we need an access token for our client.

```
curl --insecure --data "grant_type=client_credentials&username=foo@bar.com&password=secret" --user app:secret "https://localhost:9000/oauth2/token"
# You should see something like
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNTkwNDMsImlhdCI6MTQ1MDM1NTQ0MywiaXNzIjoiIiwiamlkIjoiZmMzODg4NjktZWE2MC00YzE4LWI1NmMtM2I4YmYzOTJmMzU5IiwibmJmIjoxNDUwMzU1NDQzLCJzdWIiOiJhcHAifQ.foEvIJX3hwuCJCQvIi6x31m3g1VQ0RAp6ouiiVFIs2mVM7GsD2O3aS8WxlKaxZ5P7VhbJpxTR2zg9GDSGRe-Acj26r1OVjY9QSoLIeMNg2VfA6AwpASmYhP8EOdlbyjFEK8hC14JXToWn-cT6UXE0IZxg0ANevzDSHlPnaLDemNBkxoQ1cQPIOxPOz7xZSSDZmw9rv-MNlPi6F-FNZOEig5iEyl5vzDgExr5438Qkmc5OzlLYz-RoOroFtiyoqPXp0aYEms4zaowzB4m_DrQd0cIuAKjrtlUnbvId0rOnx-PBtF6yWZfSC7_hmWwtfrmho-XFWfaawjZswRWTAgaMg",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

Hydra needs the following information to decide if a access request is allowed:
* Resource: Which resource is affected
* Permission: Which permission is requested
* Token: What access token is trying to perform this action
* Context: The context, for example the user ID.
* Header `Authorization: Bearer <token>` with a valid access token, so this endpoint can't be scanned by malicious anonymous users.

As we have said before, let's start checking if our client app *app* has the right to *create* the resource *filA.png*. The following curl request is long, you need to copy
the access token from above into both the POST body (--data "...token=...") and the Authorization header (-H "Bearer ..."):

```
# curl --insecure --data "{\"resource\": \"filA.png\", \"permission\": \"create\", \"token\": \"<client_token_from_above>\"}" -H "Authorization: Bearer <client_token_from_above>" "https://localhost:9000/guard/allowed"

# For example:
curl --insecure --data "{\"resource\": \"filA.png\", \"permission\": \"create\", \"token\": \"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNjkzOTQsImlhdCI6MTQ1MDM2NTc5NCwiaXNzIjoiIiwiamlkIjoiZWZkM2M2ODMtZTQ3Ny00ODQ4LThmZTYtZWU4NGI1YzAzZTUxIiwibmJmIjoxNDUwMzY1Nzk0LCJzdWIiOiJhcHAifQ.Q4zaiLaQvbVr9Ex3Oe9Htk-zhNsY2mtxXQgtzvnxbIbWcvF2TE_fKoVAgOGQiUiF263CNVCpKqQkMGtWcm_c1fa_2r4HYXZvOoccxHrz7foaSuLDfqcfKinlhLn_UvERT5jR9sYOA5Vw7ES1cq2WdrP17LXog9V40I0aZzmhqHXFdAv5vb4y5MdUKpaJgR_PWLBE_c12nmCRrLceSgHzVAVEyxW0BkUAK4cypIH0cz-lsSPsFZLUogQQi0oBON3FVEuXeNBxJb-Ecp3V3C5aKjrg2bs0OKeJt-ZItrzfsQF4Gsgh2irpLfF4tMN6fNDosulNT5-HuGLJGfzJzT2RYQ\"}" -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNjkzOTQsImlhdCI6MTQ1MDM2NTc5NCwiaXNzIjoiIiwiamlkIjoiZWZkM2M2ODMtZTQ3Ny00ODQ4LThmZTYtZWU4NGI1YzAzZTUxIiwibmJmIjoxNDUwMzY1Nzk0LCJzdWIiOiJhcHAifQ.Q4zaiLaQvbVr9Ex3Oe9Htk-zhNsY2mtxXQgtzvnxbIbWcvF2TE_fKoVAgOGQiUiF263CNVCpKqQkMGtWcm_c1fa_2r4HYXZvOoccxHrz7foaSuLDfqcfKinlhLn_UvERT5jR9sYOA5Vw7ES1cq2WdrP17LXog9V40I0aZzmhqHXFdAv5vb4y5MdUKpaJgR_PWLBE_c12nmCRrLceSgHzVAVEyxW0BkUAK4cypIH0cz-lsSPsFZLUogQQi0oBON3FVEuXeNBxJb-Ecp3V3C5aKjrg2bs0OKeJt-ZItrzfsQF4Gsgh2irpLfF4tMN6fNDosulNT5-HuGLJGfzJzT2RYQ" "https://localhost:9000/guard/allowed" # You should receive something like

# You should get:
{"allowed": true}
```

We could now check if our test user *foo@bar.com* has the rights to do "create" on resource "fileA.png".
*Spoiler alert:* he does not because he is not a superuser and we did not define any additional permissions.
First, we need an access token for our client and then an access token for our user:

```
# Fetch User Token
curl --insecure --data "grant_type=password&username=foo@bar.com&password=secret" --user app:secret "https://localhost:9000/oauth2/token"
# You should see something like
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNzAwODAsImlhdCI6MTQ1MDM2NjQ4MCwiaXNzIjoiIiwiamlkIjoiYTdjZDFmYWQtZTg5MS00ZDJmLWIwZmEtMzE2Zjg3MTI5ZGIyIiwibmJmIjoxNDUwMzY2NDgwLCJzdWIiOiJiY2U5M2QzMy05YjVhLTQ5MzMtOTQ3Mi1jYWRhMDE4ZGFmNjAifQ.dqUHiAJ0uoUYtV4hqhgVqYqA6PSy1cmNZQruyTpmRaCBh2RHzkijFj4F-T8xTbrFBnysTQG3LxxeXkDNq6PZBsZ4WzvUXSy1R18MayT5FWkgAi-ROQ2lHn9Isw1IgN3XWO-YOaQt9rO0gG4w_hRQ-DprMMKcUkNVC1zK_pdUpaB7cEurYF3sd7krPQjIhucPVhJqDjkAIZGG54kd28_uLqKi3eTaDrViwGLbYzmLenfTb79Hxjfd8qFd_KBQW-f1maLy0BwQNP1pVu2I_P7CBjIwEm898wTPye42CFUfVzyvB6ob4sAZM60YVwzxN_zaw_SO1160HbDI4oO-HwwPig",
  "expires_in": 3600,
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjBlNmJmNTBlLTU1N2EtNGJiYy1iZDk1LTg3ZDJkOTNkOGQ5YSJ9.JFVgu7Tf1BZJLrMbgKi0wyBKXZuHB63yKbv6_UP8TUkUgH8e9S5Gi9MhlPOnU0KyiEkh8p5Z0CMN2HQeIeYj-0p3POFxoSkY6NPZeWKsnPXzDjlJJmXWYrqgI-N-BD26MmoGXLjHt_DY3hxBX_EzHHuqVk9q-2pUAfwc0BHjSidF5EZ852I5e3J0WHbiw4KnogNRKNN-lsiIIEBSjkBxyyH85Dx4JdQZsAJVBKiXXzizWIQeQABAIutvIs5ok3T4xD8WYEiSuiHdKbPKe9bjNGX2OqW1X-eDts4RE0eHWatNQ-IafwMvi-7A0f5PSf26pSGPQ5TyvpA5qbnYAIXrMw",
  "token_type": "Bearer"
}

# Check if user is allowed to perform the action
# curl --insecure --data "{\"resource\": \"filA.png\", \"permission\": \"create\", \"token\": \"<user_account_token_from_above>\"}" -H "Authorization: Bearer <client_token_from_before>" "https://localhost:9000/guard/allowed"
# So for example:
curl --insecure --data "{\"resource\": \"filA.png\", \"permission\": \"create\", \"token\": \"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNzAwODAsImlhdCI6MTQ1MDM2NjQ4MCwiaXNzIjoiIiwiamlkIjoiYTdjZDFmYWQtZTg5MS00ZDJmLWIwZmEtMzE2Zjg3MTI5ZGIyIiwibmJmIjoxNDUwMzY2NDgwLCJzdWIiOiJiY2U5M2QzMy05YjVhLTQ5MzMtOTQ3Mi1jYWRhMDE4ZGFmNjAifQ.dqUHiAJ0uoUYtV4hqhgVqYqA6PSy1cmNZQruyTpmRaCBh2RHzkijFj4F-T8xTbrFBnysTQG3LxxeXkDNq6PZBsZ4WzvUXSy1R18MayT5FWkgAi-ROQ2lHn9Isw1IgN3XWO-YOaQt9rO0gG4w_hRQ-DprMMKcUkNVC1zK_pdUpaB7cEurYF3sd7krPQjIhucPVhJqDjkAIZGG54kd28_uLqKi3eTaDrViwGLbYzmLenfTb79Hxjfd8qFd_KBQW-f1maLy0BwQNP1pVu2I_P7CBjIwEm898wTPye42CFUfVzyvB6ob4sAZM60YVwzxN_zaw_SO1160HbDI4oO-HwwPig\"}" -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiIiLCJleHAiOjE0NTAzNjkzOTQsImlhdCI6MTQ1MDM2NTc5NCwiaXNzIjoiIiwiamlkIjoiZWZkM2M2ODMtZTQ3Ny00ODQ4LThmZTYtZWU4NGI1YzAzZTUxIiwibmJmIjoxNDUwMzY1Nzk0LCJzdWIiOiJhcHAifQ.Q4zaiLaQvbVr9Ex3Oe9Htk-zhNsY2mtxXQgtzvnxbIbWcvF2TE_fKoVAgOGQiUiF263CNVCpKqQkMGtWcm_c1fa_2r4HYXZvOoccxHrz7foaSuLDfqcfKinlhLn_UvERT5jR9sYOA5Vw7ES1cq2WdrP17LXog9V40I0aZzmhqHXFdAv5vb4y5MdUKpaJgR_PWLBE_c12nmCRrLceSgHzVAVEyxW0BkUAK4cypIH0cz-lsSPsFZLUogQQi0oBON3FVEuXeNBxJb-Ecp3V3C5aKjrg2bs0OKeJt-ZItrzfsQF4Gsgh2irpLfF4tMN6fNDosulNT5-HuGLJGfzJzT2RYQ" "https://localhost:9000/guard/allowed"

# You should get:
{"allowed": false}
```

Wow that was a lot of copy pasting, but you made it! You have used the primary features of Hydra. Obviously, this introduction did only scratch the surface and Hydra has many things to explore!
We at Ory hope that you enjoyed this tutorial and that you're going to give Hydra a try.
Hydra is not stable yet but we're working hard on getting it there. If you encounter bugs, feel free to contact us on [GitHub](https://github.com/ory-am/hydra)!

*I would like to thank [pathfinderlinden](https://www.flickr.com/photos/pathfinderlinden/7161293044/) for providing the original logo image as cc-by.*