package main

const htmlTemplate string = `
<!DOCTYPE html>
<html>
<head>
	<title>{{.Title}}</title>
	<style>
		body {
			font-size: 10px;
			font-family: sans-serif;
			padding: 0;
			margin: 0;
		}

		tbody td:nth-of-type(odd),
		tbody tr:nth-of-type(odd),
		thead th:nth-of-type(odd){
			background:rgba(178,191,196,0.5);
		}

		tbody td:nth-of-type(even),
		tbody tr:nth-of-type(even),
		thead th:nth-of-type(even){
			background:rgba(220,233,234,0.5);
		}

		table {
			border-spacing: 1;
			border-radius: 3px;
			overflow: hidden;
			width: 100%;
			margin: 0 auto;
			position: relative;
		}

		thead th {
			font-size: 1em;
			color: #fff;
			line-height: 1.2;
			font-weight: bold;
			height: 20px;
		}

		thead tr th div {
			transform: rotate(-45deg);
			transform-origin: left top -10;
		}


		thead tr {
			background-color: #36304a;
		}

		tbody td {
			white-space: nowrap;
			padding: 2px 0 2px 2px;
		}

		table {
			overflow: hidden;
		}

		tbody tr:hover {
			background-color: rgba(55, 17, 124, 0.5) !important;
		}

		td, th {
			position: relative;
		}

		tbody td:hover::after,
		tbody th:hover::after {
			content: "";
			position: absolute;
			background-color: rgba(55, 17, 124, 0.5) !important;
			left: 0;
			top: -5000px;
			height: 10000px;
			width: 100%;
			z-index: -1;
		}

		tbody td.name:hover::after {
			background-color: white !important;
		}

		tbody tr td.failed {
			background-color: rgba(220, 30, 100, 1);
		}
	</style>
</head>
<body>
	<table>
		<thead>
			<tr>
				<th class="name">Test</th>
				{{range .Headers}}
					<th><div>{{.}}</div></th>
				{{end}}
			</tr>
		</thead>
		<tbody>
			{{range .Rows}}
			<tr>
				<td class="name">{{.Name}}</td>
				{{range .Values}}
					{{if .}}
					<td class="failed"></td>
					{{else}}
					<td></td>
					{{end}}
				{{end}}
			</tr>
			{{end}}
		</tbody>
	</table>
</body>
</html>
`
