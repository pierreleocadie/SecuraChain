from jsonrpcserver import method, async_dispatch as dispatch

# Liste des clients abonnés
subscribers = set()

# Fonction RPC pour s'abonner
@method
async def subscribe(client_id):
    subscribers.add(client_id)
    return "Subscribed successfully."

# Fonction RPC pour envoyer une mise à jour aux abonnés
@method
async def send_update(message):
    for subscriber in subscribers:
        await dispatch("notify", {"client_id": subscriber, "message": message})
    return "Update sent to subscribers."

# Fonction RPC pour se désabonner
@method
async def unsubscribe(client_id):
    subscribers.discard(client_id)
    return "Unsubscribed successfully."

# Fonction RPC pour obtenir la liste des abonnés (à des fins de démonstration)
@method
async def get_subscribers():
    return list(subscribers)

# Boucle principale du serveur JSON-RPC
async def main():
    await dispatch()

if __name__ == "__main__":
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    loop.run_until_complete(main())