import http from "k6/http";

export default function() {
    let response = http.get("http://35.237.212.184:80");
};
