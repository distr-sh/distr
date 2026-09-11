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

A token consists of a key id and a secret, separated by an underscore: `distr-<key id>_<secret>`. The key id identifies the token and stays the same for its whole life, while the secret is what proves that you are allowed to use it. Distr only stores a salted hash of the secret, which is why a token is shown to you exactly once and cannot be recovered afterwards.

Both parts use digits and letters only, and the token ends in a six character checksum of everything before it. That checksum lets Distr reject a token that was mistyped or truncated on the way, and lets a secret scanner recognize one of our tokens in a repository. Tokens that were issued before secrets existed consist of the key alone and have no checksum.

## Creating a Personal Access Token

In the top right corner of the Distr web interface, click on your user icon and select **Personal Access Tokens** from the dropdown menu.
You can also directly access this page by navigating to [https://app.distr.sh/settings/access-tokens](https://app.distr.sh/settings/access-tokens) (make sure to replace `app.distr.sh` with your Distr instance URL if you selfhost).

The page lists the tokens that already exist for your user account, with a filter box above the list to find one by its label.

To create a token, click **Create token** in the top right corner. You will be prompted to enter a label, an expiry date and a role. You can leave the label and the expiry empty, but we recommend setting a descriptive label and an expiry date to keep your tokens organized and secure. All three can be changed later.

After you have entered the details, click **Create**. Distr generates the token and opens its page, where the token is displayed.

This is the only time the token will be shown to you. Make sure to copy it and store it in a secure place. Reloading the page or navigating away removes it from the screen for good.
Remember, anybody that has access to this token can authenticate with the Distr API on your behalf. Treat it like your password.

## Scoping a token's permissions

By default, a Personal Access Token operates with the same role you have in the organization. You can lower this when creating the token: the role dropdown only lets you pick a role equal to or below your own (`admin`, `read_write`, `read_only`).

The effective role on each request is the lower of the PAT's role and your current role in the organization. That means:

- A `read_only` token issued by an `admin` user can only read. It cannot push artifacts to the registry or change deployments, even though the user could.
- If your own role is later downgraded (for example from `admin` to `read_write`), every token you issued is automatically capped at the new lower role on the next request. You do not need to revoke and reissue tokens after a demotion.

We recommend creating dedicated lower-privilege tokens for automation that does not need write access, for example a `read_only` token for a CI job that only pulls artifacts from the registry.

## A token's page

Click **Details / Edit** next to a token to open its page. It shows the key id, so that you can tell which of your tokens a client is configured with, when the token was created, and its secrets with the time each of them was created and last used.

The label, the role and the expiry date can be changed here, and every change is saved right away. Lowering the role of a token takes effect on its next request, so a token you have already handed out can be restricted without reissuing it.

## Rotating a token's secret

A token can hold two secrets at the same time, so that you can replace one without ever having a moment where nothing works. Both secrets grant exactly the same access.

To rotate a secret:

1. Click **Add secret** on the token's page. Distr shows you the new token, which is the same key id with the new secret. Note it down, it is only shown once.
2. Roll the new token out everywhere the old one is in use. The "last used" timestamp of the old secret tells you whether anything is still authenticating with it.
3. Delete the old secret once its "last used" timestamp stops moving.

A token always keeps at least one secret, so the last remaining secret cannot be deleted. Delete the token itself instead.

## Securing a token created before secrets existed

Tokens created before Distr split them into a key id and a secret have no secret at all, and the list marks them with **No secret**. Such a token is its own credential, so its page shows it in full rather than only a key id.

Open it and click **Secure token** to give it a secret. Doing so replaces the token, so anything still using the old one has to be updated with the new token that is shown to you.

## Deleting Personal Access Tokens

On the same page you are also able to delete tokens. Click on the trash icon next to the token you want to delete and confirm the action.

Note that any application using this token will no longer be able to authenticate with the Distr API.
