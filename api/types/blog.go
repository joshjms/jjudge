package types

import "time"

// BlogPost represents a blog article.
type BlogPost struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Excerpt   string    `json:"excerpt"`
	AuthorID  int       `json:"author_id"`
	Author    *BlogAuthor `json:"author,omitempty"`
	Published bool      `json:"published"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BlogAuthor is the embedded author info returned with blog posts.
type BlogAuthor struct {
	Username string `json:"username"`
	Name     string `json:"name"`
}
