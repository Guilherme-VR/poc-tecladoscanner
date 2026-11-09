import 'dart:async';
import 'dart:convert';
import 'dart:io';

/// Evento enviado pelo agente Go (uma linha JSON por evento).
class EventoAgente {
  final String tipo; // pronto | tecla | serial | aviso | erro
  final String id; // aparelho de origem (USB) ou porta serial (COM3...)
  final String nome;
  final String texto; // leitura serial, ou mensagem de aviso/erro

  EventoAgente.fromJson(Map<String, dynamic> j)
    : tipo = j['tipo'] ?? '',
      id = j['id'] ?? '',
      nome = j['nome'] ?? '',
      texto = j['texto'] ?? '';
}

/// Abre o executável Go com "--agente" e lê os eventos do stdout.
class Agente {
  Process? _processo;
  final _eventos = StreamController<EventoAgente>.broadcast();
  final _saiu = Completer<int>();

  Stream<EventoAgente> get eventos => _eventos.stream;
  Future<int> get saiu => _saiu.future;

  /// Procura o agente: variável TECLADO_AGENTE (definida pelo executável
  /// único) ou "agente(.exe)" ao lado do executável do app.
  static String caminho() {
    final env = Platform.environment['TECLADO_AGENTE'];
    if (env != null && env.isNotEmpty) return env;
    final pasta = File(Platform.resolvedExecutable).parent.path;
    final nome = Platform.isWindows ? 'agente.exe' : 'agente';
    return '$pasta${Platform.pathSeparator}$nome';
  }

  Future<void> iniciar() async {
    final exe = caminho();
    if (!File(exe).existsSync()) {
      throw 'Agente não encontrado em: $exe';
    }
    final p = await Process.start(exe, ['--agente']);
    _processo = p;

    p.stdout.transform(utf8.decoder).transform(const LineSplitter()).listen((
      linha,
    ) {
      try {
        _eventos.add(EventoAgente.fromJson(jsonDecode(linha)));
      } catch (_) {}
    });
    p.stderr.transform(utf8.decoder).listen((t) {
      _eventos.add(EventoAgente.fromJson({'tipo': 'erro', 'texto': t.trim()}));
    });
    p.exitCode.then(_saiu.complete);
  }

  /// Manda um comando ao agente (uma linha JSON no stdin).
  void enviar(Map<String, dynamic> comando) {
    try {
      _processo?.stdin.writeln(jsonEncode(comando));
    } catch (_) {}
  }

  void parar() => _processo?.kill();
}
