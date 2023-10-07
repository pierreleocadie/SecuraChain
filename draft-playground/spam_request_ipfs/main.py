# script principal

import multiprocessing
from url_fetcher import get_url

urls = [
    "https://ipfs.io/ipfs/QmZGAAq9KyJVoEmXpF9vMGt3rt1WwAKtBuiQ91LhvLXG8W?filename=ipfs.drawio.png",
    "https://ipfs.io/ipfs/QmeYZpvj3hei2k58oLP9BgSdNqoXu1D4W3HJTZ2gc7aHLm?filename=Annonce.drawio.png"
]

if __name__ == "__main__":
    with multiprocessing.Pool(processes=25) as pool:
        pool.map(get_url, urls)
