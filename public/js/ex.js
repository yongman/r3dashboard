$(function(){

var WS_HOST = 'ws://' + location.host;
var openingObserver = Rx.Observer.create(function() { console.log('Opening socket'); });
var closingObserver = Rx.Observer.create(function() { console.log('Closing socket'); });

var stateSocket = Rx.DOM.fromWebSocket(
    WS_HOST +'/status', null, openingObserver, closingObserver);

var NewState = stateSocket.map(function(e){
    var state = JSON.parse(e.data);
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
    shouldComponentUpdate: function(nextProps, nextState) {
      //add appname to map,used by render sort
      _.map(nextState.apps, function(app,key){
        app.Name = key;
      });
      return !(_.isEqual(this.state.apps, nextState.apps));
    },
    render: function() {
        var ths =[];
        var n_apps = 0;
        var n_nodes = 0;
        var n_partitions = 0;
        var n_replicas = 0;
        var n_failednodes = 0;
        //sort apps by failednodes
        var sorted = [];
        _.map(this.state.apps, function(app,key){
          sorted.push(app);
        });
        n_apps = sorted.length;
        
        sorted = _.sortBy(sorted, function(app){
          return -1 * app.Exceptions;
        });
        var tds = _.map(sorted, function(app){
            var props = [];
            var href = 'http://'+location.hostname+'/'+app.Name+'/ui/cluster.html';
            props.push(<td className="positive"><a href={href} target='_blank'>{app.Name}</a></td>);
            props.push(<td>{app.TotalNodes}</td>);
            n_nodes += app.TotalNodes;
            props.push(<td>{app.Partitions}</td>);
            n_partitions += app.Partitions;
            props.push(<td>{app.Replicas}</td>);
            n_replicas += app.Replicas;
            if (app.ReplicaEqual) {
              props.push(<td>Yes</td>);
            } else {
              props.push(<td>No</td>);
            }
						if (app.ReplicaMax < 9) {
							props.push(<td>{app.ReplicaMax}G</td>);
						} else {
							props.push(<td className="negative">{app.ReplicaMax}G</td>);
						}
						if (app.ReplicaMin < 9) {
							props.push(<td>{app.ReplicaMin}G</td>);
						} else {
							props.push(<td className="negative">{app.ReplicaMin}G</td>);
						}
						if (app.ReplicaAvg < 9) {
							props.push(<td>{app.ReplicaAvg}G</td>);
						} else {
							props.push(<td className="negative">{app.ReplicaAvg}G</td>);
						}
            props.push(<td>{app.Exceptions}</td>);
            n_failednodes += app.Exceptions;
            if (!app.ReplicaEqual || app.Exceptions != 0) {
              return <tr className="center aligned negative">{props}</tr>;
            } else {
              return <tr className="center aligned">{props}</tr>;
            }
        });

        ths.push(<th>Apps({n_apps})</th>);
        ths.push(<th>TotalNodes({n_nodes})</th>);
        ths.push(<th>Partitions({n_partitions})</th>);
        ths.push(<th>Replicas({n_replicas})</th>);
        ths.push(<th>Replica Equal</th>);
				ths.push(<th>ReplicaMax</th>);
				ths.push(<th>ReplicaMin</th>);
				ths.push(<th>ReplicaAvg</th>);
        ths.push(<th>FailedNodes({n_failednodes})</th>);

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
