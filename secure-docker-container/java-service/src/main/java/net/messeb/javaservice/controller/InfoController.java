package net.messeb.javaservice.controller;

import org.springframework.boot.SpringBootVersion;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.time.LocalDateTime;
import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/api")
public class InfoController {

    @GetMapping("/info")
    public ResponseEntity<Map<String, String>> getInfo() {
        Map<String, String> info = new HashMap<>();

        // Get Spring version, Java version, and timestamp
        info.put("springVersion", SpringBootVersion.getVersion());
        info.put("javaVersion", System.getProperty("java.version"));
        info.put("timestamp", LocalDateTime.now().toString());

        // Set up custom headers
        HttpHeaders headers = new HttpHeaders();
        headers.set("Content-Type", "application/json"); // Ensure JSON header
        headers.set("Custom-Header", "CustomHeaderValue"); // Add a custom header

        // Return the response entity with body, headers, and HTTP status
        return new ResponseEntity<>(info, headers, HttpStatus.OK);
    }
}
