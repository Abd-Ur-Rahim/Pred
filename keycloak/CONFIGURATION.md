# Keycloak Configuration Guide

Step-by-step instructions for configuring Keycloak to work with the Pred application.

## Accessing Keycloak Admin Panel

### Prerequisites

Keycloak must be running:

```bash
docker compose up -d keycloak postgres
sleep 30
```

### Login to Admin Dashboard

1. Open browser: **http://localhost:8080**
2. Click **Administration Console** (bottom right corner)
3. Login with:
   - **Username:** `admin`
   - **Password:** `changeme`

---

## Creating a Client for Web Frontend

A "Client" in Keycloak represents your application. Create one so Keycloak knows how to handle login requests from your Next.js app.

### Step 1: Create New Client

1. Navigate to **Clients** (left sidebar)
2. Click **Create client** button (top right)
3. Fill in:
   - **Client ID:** `web-frontend`
   - **Name:** `Pred Web Frontend`
   - **Client type:** `OpenID Connect`
4. Click **Next**

### Step 2: Configure Capabilities

Keep defaults:
- Access Type: `Confidential` (auto-selected)
- Client authentication: `ON`
- Authorization: `OFF`
- Click **Save**

### Step 3: Set Redirect URIs

After saving, view the client details:

1. Scroll to **Login settings** section
2. Set **Valid redirect URIs:**
   ```
   http://localhost:3000/api/auth/callback/keycloak
   ```
   (This is the NextAuth callback URL — it's required)

3. Set **Valid post logout redirect URIs:**
   ```
   http://localhost:3000/login
   http://localhost:3000/
   ```

4. Click **Save**

### Step 4: Get Client Secret

1. Click **Credentials** tab
2. Under "Client secret" you'll see a generated value
3. **Copy this value**
4. Paste it in `web-frontend/.env.local`:
   ```env
   KEYCLOAK_CLIENT_SECRET=<paste-here>
   ```

---

## Creating Test Users

### Create First User

1. Navigate to **Users** (left sidebar)
2. Click **Create new user**
3. Fill in:
   - **Username:** `testuser`
   - **Email:** `test@example.com`
   - **First name:** `Test`
   - **Last name:** `User`
   - Check **Email verified** ✓
4. Click **Create**

### Set User Password

1. Go to **Credentials** tab (at the top)
2. Click **Set password**
3. Enter a password: `Test123!`
4. Check **Temporary: OFF** (user won't need to change on first login)
5. Click **Set password**
6. Confirm the dialog

### Test Login

Now test if login works:

1. Go to http://localhost:3000/login
2. Click **Login with Keycloak**
3. Enter credentials:
   - Username: `testuser`
   - Password: `Test123!`
4. You should be redirected to dashboard and see user info

---

## Advanced: Custom User Attributes (for Multi-Tenancy)

When you're ready for multi-tenant support, add custom attributes to users:

### Add Tenant ID to User

1. In **Users**, select your test user
2. Go to **Attributes** tab
3. Click **Add attribute**
4. Add:
   - **Key:** `tenant_id`
   - **Value:** `tenant-001`
5. Click **Save**

### Map Tenant ID to JWT Token

To automatically include `tenant_id` in the JWT token:

1. Navigate to **Client Scopes** (left sidebar)
2. Click **roles**
3. Go to **Mappers** tab
4. Click **Configure a new mapper** → **User Attribute**
5. Fill in:
   - **Name:** `tenant_id`
   - **User Attribute:** `tenant_id`
   - **Token Claim Name:** `tenant_id`
   - **Claim JSON Type:** `String`
6. Click **Save**

Now the JWT token will automatically include the user's `tenant_id`. Your Go services can extract and use this for multi-tenant isolation.

---

## Keycloak Realms (Optional)

By default, we use the `master` realm. Optionally, create a separate realm for your application:

### Create a New Realm

1. Hover over realm name (top-left dropdown) → **Create realm**
2. Enter **Realm name:** `prod-maintenance`
3. Click **Create**

### Switch Between Realms

Click the realm dropdown (top-left) to see all realms and switch.

**Note:** If you create a new realm, update your environment variable:
```env
KEYCLOAK_REALM=prod-maintenance
```

---

## Email Configuration (Optional)

For production, configure SMTP to send password reset emails:

1. Go to **Realm settings** (left sidebar)
2. Click **Email** tab
3. Fill in your SMTP configuration:
   - **From:** `noreply@yourdomain.com`
   - **Host:** Your SMTP server
   - **Port:** 587 (usually)
   - **Username/Password:** Your SMTP credentials
4. Click **Test connection** to verify
5. **Save**

For local development, this is optional — just reset passwords manually in admin panel.

---

## User Federation (Optional)

Connect Keycloak to an existing user database (LDAP, Active Directory):

1. Navigate to **User Federation** (left sidebar)
2. Click **Add provider**
3. Choose your provider (LDAP, Kerberos, etc.)
4. Configure connection details
5. Click **Save**

This is not needed for initial setup.

---

## Password Policy (Optional)

Set security requirements for passwords:

1. Go to **Authentication** (left sidebar)
2. Click **Password Policy** tab
3. Choose policies:
   - Minimum length: 8
   - Special characters required
   - Number required
   - etc.
4. Click **Save**

For development, use lenient policies. Tighten in production.

---

## Troubleshooting

### "Invalid client" Error

**Cause:** Client ID doesn't match  
**Fix:**
- Make sure `KEYCLOAK_CLIENT_ID` in `.env.local` matches the client ID in Keycloak (case-sensitive)
- Check client is enabled (toggle in client list)

### "Redirect URI mismatch" Error

**Cause:** Login callback URI not configured  
**Fix:**
- Go to client → **Login settings**
- Add `http://localhost:3000/api/auth/callback/keycloak` to **Valid redirect URIs**
- Save and try again

### Can't Login with Test User

**Cause:** User account disabled or password incorrect  
**Fix:**
- Go to **Users** → Select user
- Check **Enabled** is ON
- Reset password from **Credentials** tab

### Keycloak Admin Page Blank

**Cause:** Browser cache or JavaScript issue  
**Fix:**
- Hard refresh: `Cmd+Shift+R` (Mac) or `Ctrl+Shift+R` (Linux/Windows)
- Clear cookies for localhost:8080
- Try different browser

### Can't Access Admin Console

**Cause:** Wrong admin password  
**Fix:**
- Reset via Docker:
  ```bash
  docker exec keycloak /opt/keycloak/bin/kcadm.sh \
    update-user --username admin \
    --set password=newpassword123
  ```

---

## Useful Keycloak Endpoints

These endpoints are called automatically by NextAuth — no need to call manually:

| Endpoint | Purpose |
|----------|---------|
| `/.well-known/openid-configuration` | OIDC metadata |
| `/protocol/openid-connect/auth` | Authorization request |
| `/protocol/openid-connect/token` | Token exchange |
| `/protocol/openid-connect/userinfo` | Get user info |
| `/protocol/openid-connect/logout` | Logout |

Full URLs (example):
```
http://localhost:8080/realms/master/.well-known/openid-configuration
http://localhost:8080/realms/master/protocol/openid-connect/auth?...
```

---

## Production Checklist

Before deploying to production:

- [ ] Change admin password from `changeme`
- [ ] Configure SMTP for emails
- [ ] Use HTTPS (update `KEYCLOAK_URL` to use `https://`)
- [ ] Set up user federation or SSO
- [ ] Enable password policy
- [ ] Configure backup/restore
- [ ] Use production-grade Keycloak image (not `start-dev`)
- [ ] Set up monitoring and logging
- [ ] Configure TLS certificates
- [ ] Store secrets securely (not in `.env` files)

---

## Next Steps

1. ✅ Create client `web-frontend`
2. ✅ Create test user `testuser`
3. ✅ Copy client secret to `.env.local`
4. 🔲 Test login: http://localhost:3000
5. 🔲 Create more test users as needed
6. 🔲 Add custom attributes for multi-tenancy
7. 🔲 Configure email for production

See [README.md](./README.md) for quick reference.
