---
order: 3
---

# Tradding Bot

## How to use Humming Bot

:::tip
Please first create the ViteX API Key before start using humming bot.
:::

### Install Humming Bot on Ubuntu

#### Install Docker

```bash
# 1) Download Docker install script
wget https://raw.githubusercontent.com/CoinAlpha/hummingbot/development/installation/install-docker/install-docker-ubuntu.sh

# 2) Enable script permissions
chmod a+x install-docker-ubuntu.sh

# 3) Run installation
./install-docker-ubuntu.sh
```

#### Install Humming Bot

```bash
# 1) Download Hummingbot install, start, and update script
wget https://gist.githubusercontent.com/soliury/c69e352767b2521ceac83ba6775bd50f/raw/871c260483974179a97087a4146dca0c2197dc60/create.sh
wget https://gist.githubusercontent.com/soliury/43c0e649b87c7f39550aeb1f3432a835/raw/3ad918df93318d56e9f70e0647b17c87bd32fe0d/start.sh
wget https://gist.githubusercontent.com/soliury/f0f80ff3bb6b785e169a7cf7b82f4c4e/raw/2d0e1764399ebccad997d870f9c418979f329ddb/update.sh

# 2) Enable script permissions
chmod a+x *.sh

# 3) Create a hummingbot instance
./create.sh
```

#### Start Humming Bot

```bash
./start.sh
```

#### Config Humming Bot

Documentation: [Humming bot configure](https://docs.hummingbot.io/operation/client/#create-a-secure-password)


