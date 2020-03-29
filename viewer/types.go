package viewer

type Info struct {
	Id          int
	Title       string
	Author      string
	Description string
	Entries     []Entry
}

type Entry struct {
	Number            int
	Title             string
	Path              string
	ThumbnailFileName string
	ContentFileNames  []string
}
