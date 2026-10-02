RS PROTECTED SITE PACKAGE

This package adds:
- login.html
- rs-login-gate.js
- login gate injected into existing HTML pages

Cloudflare login endpoint:
https://rs-d1-api.saif362.workers.dev/api/login

IMPORTANT SECURITY NOTE:
This gate is client-side. It prevents normal navigation to the RMS HTML pages until
the Cloudflare login succeeds, but it is NOT server-side access control. Anyone who
knows the public Netlify URL can still request static files directly.

For real protection of every URL, the Netlify site must be placed behind a server-side
authentication/proxy layer or Netlify's own site access controls.

Existing RMS files were otherwise left unchanged.
