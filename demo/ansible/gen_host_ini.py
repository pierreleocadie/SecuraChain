import json
import random

# Charger les détails des instances à partir du fichier JSON
with open('instance_details.json') as f:
    instance_details = json.load(f)

total_servers = len(instance_details)

# Si aucun serveur n'est disponible pour être le repo_server, arrêter le script
if total_servers == 0:
    print("Aucun serveur disponible.")
    exit()

# Sélection aléatoire d'un serveur pour être le repo_server
repo_server_key = random.choice(list(instance_details.keys()))
repo_server = {repo_server_key: instance_details[repo_server_key]}

# Suppression du repo_server de la liste des serveurs disponibles
del instance_details[repo_server_key]

# Liste des rôles à attribuer aux serveurs restants
roles = ["bootstrap_node_servers"]

# Initialiser le dictionnaire pour stocker les serveurs sélectionnés par rôle
selected_servers = {role: [] for role in roles}

# Distribution aléatoire des serveurs restants entre les rôles
remaining_servers_keys = list(instance_details.keys())
random.shuffle(remaining_servers_keys)

for server_key in remaining_servers_keys:
    role = random.choice(roles)
    selected_servers[role].append({server_key: instance_details[server_key]})

# Générer le fichier hosts.ini
with open('hosts.ini', 'w') as f:
    f.write('[repo_server]\n')
    for key, ip in repo_server.items():
        f.write(f'{key} ansible_host={ip}\n\n')
    
    for role, servers in selected_servers.items():
        f.write(f'[{role}]\n')
        for server in servers:
            for key, ip in server.items():
                f.write(f'{key} ansible_host={ip}\n')
        f.write('\n')

print("Fichier hosts.ini généré avec succès.")
