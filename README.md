# Cryptopepes worker

This is the backend service that builds images and pushes them to google cloud storage.

Options:
```
--rpc=https://mainnet.infura.io/v3/API_KEY....
--token-address=0x84ac94f17622241f313511b629e5e98f489ad6e4
--sale-auction-address=0x28ae3df366726d248c57b19fa36f6d9c228248be
--cozy-auction-address=0xe2c43d2c6d6875c8f24855054d77b5664c7e810f
```


# Deploy

## Swap file creation

Swap required, because building takes a lot of memory

```bash
cd /var
touch swap.img
chmod 600 swap.img

# 4 GB of swap
dd if=/dev/zero of=/var/swap.img bs=1024k count=4000
mkswap /var/swap.img
swapon /var/swap.img
free

# Make swap persistent
echo "/var/swap.img    none    swap    sw    0    0" >> /etc/fstab
```

## Creation of the Dokku app

```bash
# login
eval $(ssh-agent)
ssh-add <your key path>
ssh root@<ip address>

# --- in remote machine ---

# Create worker app
dokku apps:create worker

# Configure start

dokku config:set worker DOKKU_DOCKERFILE_START_CMD="--rpc=https://mainnet.infura.io/v3/API_KEY.... \
                                                    --token-address=0x84ac94f17622241f313511b629e5e98f489ad6e4 \
                                                    --sale-auction-address=0x28ae3df366726d248c57b19fa36f6d9c228248be \
                                                    --cozy-auction-address=0xe2c43d2c6d6875c8f24855054d77b5664c7e810f"


dokku config:set worker GOOGLE_APPLICATION_CREDENTIALS="secret-key-google.json"
dokku config:set worker DATASTORE_PROJECT_ID="cryptopepe-main"
dokku config:set worker APP_PATH="/app/"


dokku docker-options:add worker build "--build-arg GOOGLE_APPLICATION_CREDENTIALS=.............."
# Replace .............. with the long base64 encoded keyfile,
#  obtained and added to your clipboard by running this locally:
base64 datastore-key-XXXXXXX.json | tr -d '\n' | xsel -pi

# back to local machine
> exit

```

## Configure local repo.

```bash
# Go to project dir
git remote add ocean-worker dokku@<ip address>:worker
```

## Pushing it live

```bash
# Deploy
git push ocean-worker master
```
