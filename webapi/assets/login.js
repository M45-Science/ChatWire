'use strict';
let token = new URLSearchParams(location.hash.slice(1)).get('token');
history.replaceState(null, '', '/login');
const button = document.querySelector('#continue');
const message = document.querySelector('#message');
if (!token) { button.disabled = true; message.textContent = 'Run /web in Discord to get a private login link.'; }
button.addEventListener('click', async () => {
  button.disabled = true;
  try {
    const response = await fetch('/auth/exchange', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({token})});
    const result = await response.json();
    if (!response.ok) throw new Error(result.error?.message || 'Sign-in failed. Run /web again.');
    token = null;
    location.replace('/');
  } catch (error) { message.textContent = error.message; button.disabled = false; }
});
