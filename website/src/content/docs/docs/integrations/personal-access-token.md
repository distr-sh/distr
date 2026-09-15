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

A token consists of a prefix and a second part, separated by an underscore: `distr-<prefix>_<second-part>`. The prefix identifies the token and stays the same for its whole life, while the second part is what proves that you are allowed to use it. Distr only stores a hash of that second part, which is why a token is shown to you exactly once and cannot be recovered afterwards.

Both parts use digits and letters only, and the token ends in a six character checksum of everything before it. That checksum lets Distr reject a token that was mistyped or truncated on the way, and lets a secret scanner recognize one of our tokens in a repository. Tokens issued before this format existed consist of the prefix alone and have no checksum.

## Creating a Personal Access Token

In the top right corner of the Distr web interface, click on your user icon and select **Personal Access Tokens** from the dropdown menu.
You can also directly access this page by navigating to [https://app.distr.sh/settings/access-tokens](https://app.distr.sh/settings/access-tokens) (make sure to replace `app.distr.sh` with your Distr instance URL if you selfhost).

The page lists the tokens that already exist for your user account, with a filter box above the list to find one by its label.

To create a token, click **Create token** in the top right corner. You will be prompted to enter a label, an expiry date and a role. You can leave the label and the expiry empty, but we recommend setting a descriptive label and an expiry date to keep your tokens organized and secure.

Only the label can be changed later. The role cannot, so that a token cannot gain access it was not created with, and neither can the expiry date: moving it would change a deadline that whatever holds the token already relies on. To keep an access token alive past that date, add a second token that expires later, as described under [rotating a token](#rotating-a-token).

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

Click **Details / Edit** next to a token to open its page. It shows when the access token was created and the role it acts under, and below that the tokens issued for it, each with its own prefix and the time it was created, expires and was last used. The prefix lets you tell which of your tokens a client is configured with, and only the beginning of it is shown, because a legacy token is nothing but its prefix.

An access token can have at most two tokens at a time. That second slot exists so that you can rotate without downtime, and **Add token** is unavailable while both are in use. An access token expires when the last token issued for it does, because every token that is still valid authenticates it.

The label can be changed here and is saved right away.

## Rotating a token

Both of an access token's two tokens can be valid at the same time, so that you can replace one without ever having a moment where nothing works. They grant exactly the same access, and two tokens of the current format also share a prefix. This is also how an access token is kept alive: the new token gets its own expiry date, and the access token stops working once both have expired.

To rotate a token:

1. Click **Add token** on the access token's page and pick the expiry date of the new one. Distr shows you the new token, which is the same prefix with a new second part. Note it down, it is only shown once.
2. Roll it out everywhere the old one is in use. The "last used" timestamp of the old token tells you whether anything is still authenticating with it.
3. Delete the old token once its "last used" timestamp stops moving.

At least one token has to remain, so the last one cannot be deleted. Delete the whole access token instead.

## Migrating a legacy token

Tokens created before Distr split them into a prefix and a second part have no second part at all, and both the list and the token's own page mark them as **Legacy**. Such a token is its own credential, which is why Distr asks you to migrate it to the format that has one.

A legacy token keeps working while you migrate, so the steps are the same as [rotating a token](#rotating-a-token): click **Add token**, roll the new one out, then delete the legacy entry once its "last used" timestamp stops moving. Nothing breaks the moment you add the token, and the legacy one stops working only when you delete it or its expiry date passes.

A legacy token counts towards the two an access token can have, so you can add exactly one token next to it and have to delete the legacy entry before adding another.

Migrating is not optional forever: a legacy token that was issued without an expiry date was given one 90 days out when your instance introduced this format, and it stops working on that date whether or not it has been migrated by then.

The two differ in the prefix they start with, because a legacy token is in the hex encoding it was issued in and the new one in the encoding the current format uses. The token's page shows the prefix per entry for exactly that reason.

## Deleting Personal Access Tokens

On the same page you are also able to delete tokens. Click on the trash icon next to the token you want to delete and confirm the action.

Note that any application using this token will no longer be able to authenticate with the Distr API.
