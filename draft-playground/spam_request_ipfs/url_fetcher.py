# url_fetcher.py
import requests

def get_url(url):
    while True:
        print(f"Fetching {url}...")
        try:
            response = requests.get(url)
            if response.status_code == 200:
                print(url, response.status_code)
                break
        except requests.exceptions.RequestException as e:
            print(f"Failed to fetch {url}: {e}")
