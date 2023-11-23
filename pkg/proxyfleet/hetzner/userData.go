package hetzner

const fedoraDefaultUserData = `#!/usr/bin/bash
dnf install -y tinyproxy
cat << EOF > /root/tinyproxy.conf
Port 8080
Listen 0.0.0.0
Timeout 600
Allow %s
MaxClients 20
StartServers 20
EOF
tinyproxy -c /root/tinyproxy.conf`

var (
	imageToUserData = map[string]string{
		"fedora-39": fedoraDefaultUserData,
		"fedora-38": fedoraDefaultUserData,
		"fedora-37": fedoraDefaultUserData,
	}
)
