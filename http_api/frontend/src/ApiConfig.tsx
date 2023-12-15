export default class ApiConfig {
    public apiEndpoint = () => "/api";
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
    public trackerEndpoint = (trackerId?: number) => {
        let str = this.apiEndpoint() + "/tracker";
        if (typeof(trackerId) !== 'undefined') {
            str += "/" + trackerId;
        }
        return str;
    }
}