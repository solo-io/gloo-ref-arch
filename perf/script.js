import http from "k6/http";

export default function() {
    let response = http.get("http://35.196.131.38:80");
};
