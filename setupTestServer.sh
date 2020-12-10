mkdir ./server
wget -O ./server/serverArchive.zip https://terraria.org/system/dedicated_servers/archives/000/000/042/original/terraria-server-1412.zip
cd ./server/ || exit
unzip serverArchive.zip
cp -r ./1412/Linux/. ./
chmod +x TerrariaServer.bin.x86_64
wget https://www.dropbox.com/s/g8byen9hyjf5d4c/Texas.wld
cp Texas.wld Texas.wld.bak
rm serverArchive.zip