package handlers

import "encoding/xml"

type AtomAuthor struct {
	Name string `xml:"name"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
}

type AtomContent struct {
	Type string `xml:"type,attr"`
	Text string `xml:",chardata"`
}

type AtomEntry struct {
	ID        string      `xml:"id"`
	Link      AtomLink    `xml:"link"`
	Title     string      `xml:"title"`
	Author    AtomAuthor  `xml:"author,omitempty"`
	Updated   string      `xml:"updated"`
	Summary   string      `xml:"summary,omitempty"`
	Content   AtomContent `xml:"content,omitempty"`
	Published string      `xml:"published,omitempty"`
}

type AtomFeed struct {
	ID      string      `xml:"id"`
	Link    AtomLink    `xml:"link"`
	Title   string      `xml:"title"`
	Author  AtomAuthor  `xml:"author,omitempty"`
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Updated string      `xml:"updated"`
	Entries []AtomEntry `xml:"entry"`
}
