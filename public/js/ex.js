$(function(){

var WS_HOST = 'ws://' + location.host;
var openingObserver = Rx.Observer.create(function() { console.log('Opening socket'); });
var closingObserver = Rx.Observer.create(function() { console.log('Closing socket'); });

var stateSocket = Rx.DOM.fromWebSocket(
    WS_HOST +'/status', null, openingObserver, closingObserver);

var NewState = stateSocket.map(function(e){
    var state = JSON.parse(e.data);
    console.log(state);
    return state;
});


var Example = React.createClass({
    getInitialState: function () {
        return {
            apps: {}
        };
    },
    componentDidMount: function() {
        var self = this;
        NewState.subscribe(
            function(obj) {
                self.setState({apps: obj});
            },
            function (e) {
                console.log('Error: ', e);
            },
            function () {
                console.log('Closed');
            }
        );
    },
    render: function() {
        var ths =[];
        ths.push(<th>App</th>);
        ths.push(<th>TotalNodes</th>);
        ths.push(<th>Partitions</th>);
        ths.push(<th>Replicas</th>);
        ths.push(<th>Replica Equal</th>);
        ths.push(<th>FailedNodes</th>);
        var tds = _.map(this.state.apps, function(app,key){
            var props = [];
            props.push(<td className="positive">{key}</td>);
            props.push(<td>{app.TotalNodes}</td>);
            props.push(<td>{app.Partitions}</td>);
            props.push(<td>{app.Replicas}</td>);
            if (app.ReplicaEqual) {
              props.push(<td>Yes</td>);
            } else {
              props.push(<td>No</td>);
            }
            props.push(<td>{app.Exceptions}</td>);
            if (!app.ReplicaEqual || app.Exceptions != 0) {
              return <tr className="center aligned negative">{props}</tr>;
            } else {
              return <tr className="center aligned">{props}</tr>;
            }
        });

        return (
            <div className="ui vertical stripe quote segment">
            <table className="ui celled table">
                <thead>
                    <tr>{ths}</tr>
                </thead>
                <tbody>
                {tds}
                </tbody>
            </table>
            </div>
        );
    }
});

React.render(
    <Example />,
    document.getElementById('content')
);

});
