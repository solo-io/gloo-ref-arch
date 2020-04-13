import http from "k6/http";

export default function() {
    var params = {
      headers: {
        'stage': 'canary',
      },
    };
    let response = http.get("http://35.227.127.150:80");
};
