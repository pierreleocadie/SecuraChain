import requests
import json

# Configuration du serveur JSON-RPC
server_url = "http://localhost:5000"  # Remplacez par l'URL du serveur JSON-RPC

# Fonction pour envoyer une requête JSON-RPC
def send_request(method, params=None):
    payload = {
        "jsonrpc": "2.0",
        "method": method,
        "params": params if params is not None else [],
        "id": 1,
    }
    response = requests.post(server_url, json=payload)
    return response.json()

# Exemple d'abonnement
response = send_request("subscribe", ["client123"])
print(response)

# Exemple de réception de mises à jour
while True:
    message = input("Entrez un message à envoyer aux abonnés (ou 'exit' pour quitter) : ")
    if message.lower() == "exit":
        break
    response = send_request("send_update", [message])
    print(response)

# Exemple de désabonnement
response = send_request("unsubscribe", ["client123"])
print(response)

# Exemple pour obtenir la liste des abonnés (à des fins de démonstration)
response = send_request("get_subscribers")
print("Liste des abonnés :", response)
