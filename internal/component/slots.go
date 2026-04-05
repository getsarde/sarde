package component

// Built-in component slot names. Each slot has a default template
// compiled into the binary and can be overridden by themes or users.
const (
	SlotHead                  = "Head"
	SlotHeader                = "Header"
	SlotSiteTitle             = "SiteTitle"
	SlotGlobalNav             = "GlobalNav"
	SlotSearch                = "Search"
	SlotThemeToggle           = "ThemeToggle"
	SlotSidebar               = "Sidebar"
	SlotTableOfContents       = "TableOfContents"
	SlotMobileTableOfContents = "MobileTableOfContents"
	SlotBreadcrumbs           = "Breadcrumbs"
	SlotPagination            = "Pagination"
	SlotPageTitle             = "PageTitle"
	SlotContentPanel          = "ContentPanel"
	SlotFooter                = "Footer"
	SlotEditLink              = "EditLink"
	SlotLastUpdated           = "LastUpdated"
	SlotFallbackNotice        = "FallbackNotice"
)

// AllSlots returns all built-in component slot names.
func AllSlots() []string {
	return []string{
		SlotHead, SlotHeader, SlotSiteTitle, SlotGlobalNav,
		SlotSearch, SlotThemeToggle, SlotSidebar, SlotTableOfContents,
		SlotMobileTableOfContents, SlotBreadcrumbs, SlotPagination,
		SlotPageTitle, SlotContentPanel, SlotFooter, SlotEditLink,
		SlotLastUpdated, SlotFallbackNotice,
	}
}
