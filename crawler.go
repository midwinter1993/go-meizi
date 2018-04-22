package main

import (
    "fmt"
    "io"
    "io/ioutil"
    "net/http"
    "os"
    "path"
    "regexp"
    "strconv"
    "time"
)

type ImageInfo struct {
    img_page_url string
    img_fpath string
}

type AlbumInfo struct {
    album_url string
    album_dir string
    album_title string
    album_nr_img int
}

func (info *AlbumInfo)String() string {
    return fmt.Sprintf("url: %s\ndir: %s\ntitle: %s\n#img: %d",
                       info.album_url,
                       info.album_dir,
                       info.album_title,
                       info.album_nr_img)
}

var IMG_DIR = "./images"
var SITE_URL = "http://www.mzitu.com"
var TOT_NR_TO_DOWNLOAD = 10000

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
        fmt.Println("Create Dir:", IMG_DIR)
        create_dir(IMG_DIR)
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
    r, _ := regexp.Compile(pat)
    matches := r.FindAllStringSubmatch(html, -1)

    max_nr := 0
    for _, m := range matches {
        i, _ := strconv.Atoi(m[1])
        if max_nr < i {
            max_nr = i
        }
    }
    return max_nr
}

func extract_album_info(album_url string) *AlbumInfo {
    html := get_html(album_url)

    title := album_title(html)
    if is_downloaded(title) {
        return nil
    }
    add_downloaded(title)

    album_dir := path.Join(IMG_DIR, title)
    max_nr := album_tot_nr(album_url, html)

    info := new(AlbumInfo)
    info.album_url = album_url
    info.album_dir = album_dir
    info.album_title = title
    info.album_nr_img = max_nr
    return info
}

//===============================================
// Download an image from one page containing
// an image
//-----------------------------------------------
func download_img(img_page_url string, fpath string) {
    img_url := extract_img_url(get_html(img_page_url))
    download_file(img_url, fpath)
}

func extract_img_url(html string) string {
    if html == "" {
        fmt.Println("html is empry")
        return ""
    }
    r, _ := regexp.Compile("<img src=\"(.*)\" alt.*/>")
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
func img_crawler(img_info_chan chan ImageInfo) {
    nr_downloaded := 0

    for i := 1; i < TOT_NR_TO_DOWNLOAD/30; i += 1 {
        page_url := fmt.Sprintf("%s/page/%d/", SITE_URL, i)

        html := get_html(page_url)
        album_urls := extract_album_url(html)
        for _, u := range album_urls {
            // fmt.Println(u)
            info := extract_album_info(u)
            if info == nil {
                continue
            }
            fmt.Println(info)

            for i := 1; i <= info.album_nr_img; i += 1 {
                img_page_url := fmt.Sprintf("%s/%d", info.album_url, i)

                fname := fmt.Sprintf("%d.jpg", i)
                fpath := path.Join(info.album_dir, fname)

                create_dir(info.album_dir)

                nr_downloaded += 1
                img_info_chan <- ImageInfo{img_page_url, fpath}
            }
        }
        if nr_downloaded > TOT_NR_TO_DOWNLOAD {
            break
        }
        time.Sleep(1000 * time.Millisecond)
    }
    close(img_info_chan)
}

func main() {
    load_downloaded()

    img_info_chan := make(chan ImageInfo, 300)
    go img_crawler(img_info_chan)

    for i := range img_info_chan {
        download_img(i.img_page_url, i.img_fpath)
        time.Sleep(1000 * time.Millisecond)
    }
}
