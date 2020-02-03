package main

import ("net/http"
        "encoding/json"
        "strings"
        "time"
        "log"
        "fmt"
)

// Code for version 1
/* type weatherData struct {
    Name string `json:"name"`
    Main struct {
        Kelvin float64 `json:"temp"`
    } `json:"main"`
}

func main(){
    http.HandleFunc("/hello", hello)
    http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
    
    city := strings.SplitN(r.URL.Path, "/", 3)[2]
    data, err := query(city)

    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }    

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    json.NewEncoder(w).Encode(data)
    })

    http.ListenAndServe(":8080", nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello!"))
}

func query(city string) (weatherData, error) {
    resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=641916f8c2766e292f418aefc7ceff7b&q=" + city)
    if err != nil {
        return weatherData{}, err
    }
    defer resp.Body.Close()
    var d weatherData

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return weatherData{}, err
    }

    return d, nil
} */

func main() {
    mw := multiWeatherProvider{
        openWeatherMap{},
        darkSky{},
        //weatherUnderground{apiKey: "KEY"},   //Uncomment this with a valid API key
    }
    
    http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
        begin := time.Now()
        city := strings.SplitN(r.URL.Path, "/", 3)[2]
        temp, err := mw.temperature(city)

        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        json.NewEncoder(w).Encode(map[string] interface{}{
            "city": city,
            "temp": temp,
            "took": time.Since(begin).String(),
        })
    })
    
    http.ListenAndServe(":8080", nil)
}


type weatherProvider interface {
    temperature(city string) (float64, error)
}

type multiWeatherProvider []weatherProvider

//Code for version 2
/* func (w multiWeatherProvider) temperature(city string) (float64, error) {
    sum := 0.0
    for _,provider := range w {
        k,err := provider.temperature(city)
        if err != nil {
            return 0, nil
        }
        sum += k
    }
    return sum/float64(len(w)), nil
} */

func (w multiWeatherProvider) temperature(city string) (float64, error) {
    temps := make(chan float64, len(w))
    errs := make(chan error, len(w))

    for _,provider := range w {
        go func(p weatherProvider) {
            k, err := p.temperature(city)
            if err != nil {
                errs <- err
                return
            }
            temps <- k
        }(provider)
    }

    sum := 0.0

    for i:=0; i<len(w); i++ {
        select {
        case temp := <-temps:
            sum += temp
        case err := <-errs:
            return 0, err
        }
    }

    return sum/float64(len(w)), nil
}

type openWeatherMap struct {}

func (w openWeatherMap) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=641916f8c2766e292f418aefc7ceff7b&q=" + city)
    if err != nil {
        return 0, nil
    }
    
    defer resp.Body.Close()

    var d struct {
        Main struct {
            Kelvin float64 `json:"temp"`
        } `json:"main"`
        Coord struct {
            Lon float64 `json:"lon"`
            Lat float64 `json:"lat"`
        } `json:"coord"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, nil
    }

    log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)
    return d.Main.Kelvin, nil
}

type darkSky struct {}

func (w darkSky) temperature(city string) (float64, error) {
    owm, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=641916f8c2766e292f418aefc7ceff7b&q=" + city)
    if err != nil {
        return 0, nil
    }

    defer owm.Body.Close()

    var c struct {
        Coord struct {
            Lon float64 `json:"lon"`
            Lat float64 `json:"lat"`
        } `json:"coord"`
    }
    
    if err := json.NewDecoder(owm.Body).Decode(&c); err != nil {
        return 0, nil
    }

    resp, err := http.Get("https://api.darksky.net/forecast/07912fd6985dfd172c5faeaaf13b582c/" + fmt.Sprintf("%.2f", c.Coord.Lat) + "," + fmt.Sprintf("%.2f", c.Coord.Lon))
    if err != nil {
        return 0, nil
    }

    defer resp.Body.Close()

    var d struct {
        Currently struct {
            Temp float64 `json:"temperature"`
        } `json:"currently"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, nil
    }

    temp := (((d.Currently.Temp - 32)/9.0)*5.0) + 273.15

    log.Printf("darkSky: %s: %.2f", city, temp)
    return temp, nil
}

type weatherUnderground struct {
    apiKey string
}

func (w weatherUnderground) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.wunderground.com/api/" + w.apiKey + "/conditions/q/" + city + ".json")
    if err != nil {
        return 0, nil
    }

    defer resp.Body.Close()

    var d struct {
        Observation struct {
            Celcius float64 `json:"temp_c"`
        } `json:"current_observation"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, nil
    }

    kelvin := d.Observation.Celcius + 273.15

    log.Printf("weatherUnderground: %s: %.2f", city, kelvin)
    return kelvin, nil
}
