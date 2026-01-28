package handlers

type AtomAuthor struct {
	Name string `xml:"name"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
}
