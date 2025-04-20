#!/bin/bash
#Check if parameters provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <url> <service_name> <image_wth_tag>"
    exit 1
fi

URL="$1"
SERVICE_NAME="$2"
IMAGE_WTH_TAG="$3"
SUCCESS=true
MAX_ATTEMPTS=3

# Function to check URL availability
check_url() {
    # Using curl with timeout of 10 seconds, silent mode
    # --fail option ensures curl returns non-zero status on HTTP errors (4xx, 5xx)
    if curl --fail --silent --head --max-time 10 "$URL" > /dev/null; then
        return 0  # Success
    else
        return 1  # Failure
    fi
}

# Deploy with no traffic and tag green
gcloud run deploy $SERVICE_NAME --image $IMAGE_WITH_TAG  --no-traffic --tag green

# Perform three checks
for attempt in $(seq 1 $MAX_ATTEMPTS); do
    echo "Check $attempt of $MAX_ATTEMPTS..."
    if ! check_url; then
        SUCCESS=false
        echo "Check $attempt failed!"
        break
    fi
    echo "Check $attempt passed."
    
    # Small delay between checks
    if [ $attempt -lt $MAX_ATTEMPTS ]; then
        sleep 1
    fi
done

# Print final result
if [ "$SUCCESS" = true ]; then
    echo "SUCCESS: All URL checks passed for $URL"
    echo "Switching traffic to green revision"
    gcloud run services update-traffic $SERVICE_NAME --to-tags green=100
    exit 0
else
    echo "FAILURE: URL check failed for $URL"
    echo "Rebuild and try again"
    exit 1
fi
