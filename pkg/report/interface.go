package report

type ArchivedReportStore interface {
	RetrieveArchivedReportSummaries() ([]*ArchivedReportData, error)
	RetrieveArchivedReport(string) (*ArchivedReportData, error)
	StoreArchivedReport(*ArchivedReportData) error
}
