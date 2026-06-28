#! bin/sh

case "$ARCH" in
    aarch64|arm*)
        echo "Detected ARM architecture ($ARCH)"
        echo "ARM is not supported by this installer, please build the client from source"
        exit 1
        ;;
esac

START_DIR=$(pwd)
cd $HOME/.local/bin/

echo "Downloading client..."

# add last commit to the url to bust the cache
curl -o pointclient https://cdn.lu2000luk.com/pointhole/client/client?commit=8b1dc53
chmod +x pointclient

echo "Client installed to $HOME/.local/bin/pointclient"

cd $START_DIR

echo "You can now run the client with the command: pointclient"