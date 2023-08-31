export default class ApiConfig {
    public apiEndpoint = () => "http://127.0.0.1:8000/api";
    public torrentEndpoint = (torrentId?: number) => {
        let str = this.apiEndpoint() + "/torrent";
        if (typeof(torrentId) !== 'undefined') {
            str += "/" + torrentId;
        }
        return str;
    }
    public fileEndpoint = (torrentId: number, fileId?: number | string) => {
        let str = this.apiEndpoint() + "/torrent/" + torrentId + "/file";
        if (typeof(fileId) !== 'undefined') {
            str += "/" + fileId;
        }
        return str;
    }
}