kill -9 $(ps aux | grep 'popcorn' | awk '{print $2}')
