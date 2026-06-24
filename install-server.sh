#! bin/sh

case "$ARCH" in
    aarch64|arm*)
        echo "Detected ARM architecture ($ARCH)"
        echo "ARM is not supported by this installer, please build the server from source"
        exit 1
        ;;
esac

START_DIR=$(pwd)
cd $HOME/.local/bin/

echo "Downloading server..."

# add last commit to the url to bust the cache
curl -o pointserver https://cdn.lu2000luk.com/pointhole/server/server?commit=e80d169
chmod +x pointserver

echo "Server installed to $HOME/.local/bin/pointserver"

cd $START_DIR

echo "You can now run the server with the command: pointserver"