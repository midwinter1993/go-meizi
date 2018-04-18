package main

import (
    "fmt"
    "net/http"
    "io/ioutil"
    "regexp"
    "os"
    "io"
    "strconv"
    "path"
    "time"
)

var IMG_DIR = "./images"

var downloaded = make(map[string]bool)

func add_downloaded(album_title string) {
    downloaded[album_title] = true
}

func is_downloaded(album_title string) bool {
    if _, ok := downloaded["foo"]; ok {
        return true
    }
    return false
}

func load_downloaded() {
    files, err := ioutil.ReadDir(IMG_DIR)
    if err != nil {
        fmt.Println(err)
    }

    for _, f := range files {
        // fmt.Println(f.Name())
        add_downloaded(f.Name())
    }
}

//===============================================
// Utils
//-----------------------------------------------
func create_dir(dir string) {
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        os.Mkdir(dir, os.ModePerm)
    }
}

func get_html(url string) string {
    resp, err := http.Get(url)

    if err != nil {
        fmt.Println("http transport error is:", err)
        return ""
    }

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("read error is:", err)
        return ""
    }
    defer resp.Body.Close()

    return string(body)
}

func extract_album_url(html string) []string {
    var urls []string

    pat := "<li><a href=\"(.+?)\" target=\"_blank\">.+?</a>"
    // fmt.Println(pat)
    r, _ := regexp.Compile(pat)
    matches := r.FindAllStringSubmatch(html, -1)

    // fmt.Println(matches)
    for _, m := range matches {
        urls = append(urls, m[1])
    }

    return urls
}

//===============================================
// Download an album containing multiple images
//-----------------------------------------------
func album_title(html string) string {
    if html == "" {
        fmt.Println("html is empry")
        return ""
    }
    pat := "<h2 class=\"main-title\">(.+)</h2>"
    r, _ := regexp.Compile(pat)
    match := r.FindStringSubmatch(html)
    // fmt.Println(match)
    return match[1]
}

func album_tot_nr(url string, html string) int {
    if html == "" {
        fmt.Println("html is empry")
        return -1
    }
    pat := fmt.Sprintf("<a href='%s/(\\d+)'>", url)
    fmt.Println(pat)
    r, _ := regexp.Compile(pat)
    matches := r.FindAllStringSubmatch(html, -1)
    // fmt.Println(matches)
    max_nr := 0
    for _, m := range matches {
        i, _ := strconv.Atoi(m[1])
        if max_nr < i {
            max_nr = i
        }
    }
    return max_nr
}

func download_album(album_url string) {
    html := get_html(album_url)
    // fmt.Println(html)

    title := album_title(html)
    if is_downloaded(title) {
        return
    }
    add_downloaded(title)
    fmt.Println(title)

    album_dir := path.Join(IMG_DIR, title)
    create_dir(album_dir)

    max_nr := album_tot_nr(album_url, html)
    for i := 1; i <= max_nr; i += 1 {
        img_page_url := fmt.Sprintf("%s/%d", album_url, i)

        fname := fmt.Sprintf("%d.jpg", i)
        fpath := path.Join(album_dir, fname)

        fmt.Println(img_page_url, fpath)

        download_img(img_page_url, fpath)
        time.Sleep(1000 * time.Millisecond)
    }
}

//===============================================
// Download an image from one page containing
// an image
//-----------------------------------------------
func download_img(url string, fpath string) {
    img_url := extract_img_url(get_html(url))
    // fmt.Println(img_url)
    download_file(img_url, fpath)
}

func extract_img_url(html string) string {
    if html == "" {
        fmt.Println("html is empry")
        return ""
    }
    r, _ := regexp.Compile("<img src=\"(.*)\" alt.*/>")
    // fmt.Println(r.FindStringSubmatch(html))
    match := r.FindStringSubmatch(html)
    return match[1]
}

func download_file(url string, fpath string) (err error) {
    out, err := os.Create(fpath)
    if err != nil {
        return err
    }
    defer out.Close()

    client := &http.Client{}
    req, _ := http.NewRequest("GET", url, nil)
    // For anti-crawler
    req.Header.Set("Referer", url)

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("bad status: %s", resp.Status)
    }

    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return err
    }
    return nil
}

//===============================================

func main() {
    load_downloaded()
    site_url := "http://www.mzitu.com"
    // download_album(url)
    for i := 1; i < 3; i += 1 {
        page_url := fmt.Sprintf("%s/page/%d/", site_url, i)

        html := get_html(page_url)
        album_urls := extract_album_url(html)
        for _, u := range album_urls {
            fmt.Println(u)
            // download_album(u)
        }
    }
}
