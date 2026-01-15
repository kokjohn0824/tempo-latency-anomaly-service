package service

import (
    "context"
    "sort"
    "strings"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// ListAvailable provides functionality to list available services and endpoints.
type ListAvailable struct {
    store      store.Store
    minSamples int
}

// NewListAvailable creates a new ListAvailable service.
func NewListAvailable(st store.Store, minSamples int) *ListAvailable {
    return &ListAvailable{
        store:      st,
        minSamples: minSamples,
    }
}

// GetAvailableServices retrieves all services and endpoints with sufficient samples.
func (s *ListAvailable) GetAvailableServices(ctx context.Context) (*domain.AvailableServicesResponse, error) {
    // Get all baseline keys with sufficient samples
    keys, err := s.store.ListBaselineKeys(ctx, s.minSamples)
    if err != nil {
        return nil, err
    }

    // Parse keys and group by service and endpoint
    // Key format: base:{service}|{endpoint}|{hour}|{dayType}
    serviceMap := make(map[string]map[string][]string) // service -> endpoint -> buckets

    for _, key := range keys {
        // Remove "base:" prefix
        if !strings.HasPrefix(key, "base:") {
            continue
        }
        keyPart := strings.TrimPrefix(key, "base:")
        
        // Split by "|"
        parts := strings.Split(keyPart, "|")
        if len(parts) < 4 {
            continue
        }

        service := parts[0]
        endpoint := strings.Join(parts[1:len(parts)-2], "|") // Handle endpoints with "|" in name
        hour := parts[len(parts)-2]
        dayType := parts[len(parts)-1]
        bucket := hour + "|" + dayType

        if serviceMap[service] == nil {
            serviceMap[service] = make(map[string][]string)
        }
        serviceMap[service][endpoint] = append(serviceMap[service][endpoint], bucket)
    }

    // Convert map to response structure
    // Initialize as empty slice instead of nil to ensure JSON serialization as []
    services := make([]domain.ServiceEndpoint, 0)
    totalEndpoints := 0

    // Sort services for consistent output
    var sortedServices []string
    for service := range serviceMap {
        sortedServices = append(sortedServices, service)
    }
    sort.Strings(sortedServices)

    for _, service := range sortedServices {
        endpoints := serviceMap[service]
        
        // Sort endpoints for consistent output
        var sortedEndpoints []string
        for endpoint := range endpoints {
            sortedEndpoints = append(sortedEndpoints, endpoint)
        }
        sort.Strings(sortedEndpoints)

        for _, endpoint := range sortedEndpoints {
            buckets := endpoints[endpoint]
            sort.Strings(buckets) // Sort buckets too
            
            services = append(services, domain.ServiceEndpoint{
                Service:  service,
                Endpoint: endpoint,
                Buckets:  buckets,
            })
            totalEndpoints++
        }
    }

    return &domain.AvailableServicesResponse{
        TotalServices:  len(serviceMap),
        TotalEndpoints: totalEndpoints,
        Services:       services,
    }, nil
}
