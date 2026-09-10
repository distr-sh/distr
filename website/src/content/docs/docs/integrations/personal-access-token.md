---
title: Creating a Personal Access Token
description: Generate secure access tokens to authenticate with Distr's Registry, API and SDK.
slug: docs/integrations/personal-access-token
sidebar:
  label: Personal Access Tokens
  order: 4
---

In integration scenarios, you need to authenticate with the Distr API using a Personal Access Token (PAT).
This applies for any kind of integration, no matter whether you are using the Distr SDK or interacting with the API directly.

A Personal Access Token is a unique string that you generate in the Distr web interface. It is directly associated with the user who created it, and with the organization it was created in. It cannot be used to access data of other organizations of the same user.

A token consists of a key and a secret, separated by an underscore: `distr-<key>_<secret>`. The key identifies the token, while the secret is what proves that you are allowed to use it. Distr only stores a salted hash of the secret, which is why a token is shown to you exactly once and cannot be recovered afterwards.

Both parts use digits and letters only, and the token ends in a six character checksum of everything before it. That checksum lets Distr reject a token that was mistyped or truncated on the way, and lets a secret scanner recognize one of our tokens in a repository. Tokens that were issued before secrets existed consist of the key alone and have no checksum.

## Creating a Personal Access Token

In the top right corner of the Distr web interface, click on your user icon and select **Personal Access Tokens** from the dropdown menu.
You can also directly access this page by navigating to [https://app.distr.sh/settings/access-tokens](https://app.distr.sh/settings/access-tokens) (make sure to replace `app.distr.sh` with your Distr instance URL if you selfhost).

The page will show you a list of already granted access tokens that are associated with your user account.

On the top right corner, click on the **Create token** button.

![Personal Access Tokens](../../../../assets/docs/integrations/pat_create.png)

You will be prompted to enter a name, an expiry date, and a role for the token.
You can leave the name and expiry empty, but we recommend setting a descriptive label and an expiry date to keep your tokens organized and secure.

![Personal Access Tokens](../../../../assets/docs/integrations/pat_details.png)

After you have entered the details, click on the **Create** button. Distr generates the token and opens its page, where the token is displayed.

## Scoping a token's permissions

By default, a Personal Access Token operates with the same role you have in the organization. You can lower this when creating the token: the role dropdown only lets you pick a role equal to or below your own (`admin`, `read_write`, `read_only`).

The effective role on each request is the lower of the PAT's role and your current role in the organization. That means:

- A `read_only` token issued by an `admin` user can only read. It cannot push artifacts to the registry or change deployments, even though the user could.
- If your own role is later downgraded (for example from `admin` to `read_write`), every token you issued is automatically capped at the new lower role on the next request. You do not need to revoke and reissue tokens after a demotion.

We recommend creating dedicated lower-privilege tokens for automation that does not need write access, for example a `read_only` token for a CI job that only pulls artifacts from the registry.

This is the only time the token will be shown to you. Make sure to copy it and store it in a secure place.
Remember, anybody that has access to this token can authenticate with the Distr API on your behalf. Treat it like your password.

![Personal Access Tokens](../../../../assets/docs/integrations/pat_output.png)

## Rotating a token's secret

A token can hold two secrets at the same time, so that you can replace one without ever having a moment where nothing works. Both secrets grant exactly the same access. Click **Details** next to a token to open it, which shows the token's key, so that you can tell which of your tokens a client is configured with, and for each secret when it was created and when it was last used.

To rotate a secret:

1. Click **Add secret** on the token's page. Distr shows you the new token, which is the same key with the new secret. Note it down, it is only shown once.
2. Roll the new token out everywhere the old one is in use. The "last used" timestamp of the old secret tells you whether anything is still authenticating with it.
3. Delete the old secret once its "last used" timestamp stops moving.

A token always keeps at least one secret, so the last remaining secret cannot be deleted. Delete the token itself instead.

## Securing a token created before secrets existed

Tokens created before Distr split them into a key and a secret have no secret at all, and the list marks them with **No secret**. Open such a token and click **Secure token** to give it a secret. Doing so replaces the token, so anything still using the old one has to be updated with the new token that is shown to you.

## Deleting Personal Access Tokens

On the same page you are also able to delete tokens. Click on the trash icon next to the token you want to delete and confirm the action.

Note that any application using this token will no longer be able to authenticate with the Distr API.
