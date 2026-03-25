So our Clerk auth implementation fell drastically short.

ISSUES WE EXPERIENCED:
- Clerk forces their own UI into our application, it breaks the unique feel and touch of our app
- Clerk felt very vendor-in like
- Clerk has upgrade to PRO features which is not what I want when I'm making a cloud application

What I liked about Clerk:
- Easy auth support: it was easy to drop-in and play.
- The DX was modern and nice

what I truly hated about clerk:
- I mentioned it above already, but there is more
- For every UI that felt short, I had to write my own and that felt very hacky
- I would be forced to use their SDKs and upgrade to PRO for more features.
- They limit Organizations to X amount of users, unless you upgrade.

Here is what I am thinking of doing next:
- Something minimal, but production-ready.
- Handles JWT, Multiple session sign-in, SSO, good security
- Doesn't have a vendor lock-in
- Can scale
- It has a decent DX
- Doesn't force prebuilt UIs
- Don't have to pay for PRO
- Is cheap and maintainable at scale

What would be nice to have, but not extremely important:
- Sending emails
- Organizations

Zitadel

Okay, let's plan out the migration.

What we need:

- Username, Email, Password, Optional Phone Number

Later, when the person is signed in:
- If they used a SSO provider (Google, Github, etc..)
- We will first need to prompt them to actually set their username.

If they used credentials (Password), we send them an activation email to activate their account.
If they used an SSO provider, we will just send them a warm welcome email to their email address.

Next, we need to obviously handle all edgecases:

- Forgot password
- Change username
- Change password
- Change email address
- Delete account

Next, we can focus on Organizations and multi-tenants:

- A user can create an organization as part of their account.
- A user can delete, change, modify that organization and can invite different members aboard.

Please note if missing anything. 


  1. User table — You don't have one. Clerk held all user data. You'll need a users table in your DB that syncs with Zitadel.
  2. User-to-org membership — Currently the org is embedded in Clerk's JWT claims (ActiveOrganizationID). You'll need a memberships table to track who belongs to which org with what
  role.
  3. Email verification flow — Not just "send an activation email." You need a verification token, expiry, and an endpoint to handle the click-through.
  4. Account linking — User signs up with email/password, later adds Google SSO (or vice versa). Same email, different auth methods. Zitadel handles this, but you need to decide your
  linking policy.
  5. Session management — You mentioned multi-session. Need to decide: multiple sessions per device? Multiple org sessions? Zitadel supports this but it's a design decision.
  6. Rate limiting on auth endpoints — Login, registration, password reset are prime brute-force targets. Your current rate limiting is on API routes only.
  7. Audit trail — Org member invited, role changed, member removed. Important for multi-tenant SaaS.

- Multiple devices can be logged in, they're tracked inside of the Dashboard where the person who owns the account can delete a session in there if no longer needed. It will provide information about the session.

- For ratelimiting, we shall use Redis, which we already have in our codebase well implemented.

- Audit trail: Yes, that's important.


Keep in mind: a user can have a personal account, and can make an organization. This will be used both for solo devs and teams.

Now, since we're deploying another container on Cloud Run, shall the auth code live in that container, or separate container, or in the API? 