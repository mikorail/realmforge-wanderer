#!/bin/sh
# Ensures Node >= 22 (from available nvm) is used for Vite 8 / SvelteKit
if [ -d "$HOME/.nvm/versions/node/v24.14.0/bin" ]; then
	export PATH="$HOME/.nvm/versions/node/v24.14.0/bin:$PATH"
elif [ -s "$HOME/.nvm/nvm.sh" ]; then
	. "$HOME/.nvm/nvm.sh"
	nvm use 24 >/dev/null 2>&1 || nvm use >/dev/null 2>&1
fi
exec "$@"
