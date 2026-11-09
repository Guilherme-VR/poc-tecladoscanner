import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'agente.dart';

void main() => runApp(const App());

class App extends StatelessWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Teclado x Scanner',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(colorSchemeSeed: Colors.blue, useMaterial3: true),
      home: const Tela(),
    );
  }
}

const maxScanners = 2;
const corTeclado = Color(0xFF1E66D0);
const corSerial = Colors.blueGrey;
const coresScanner = [Color(0xFFE07B00), Color(0xFF8E24AA)];

/// Scanner cadastrado. O ID só serve para reconhecer o aparelho: porta
/// serial (COM3, /dev/ttyACM0) ou o ID único do aparelho USB.
class Scanner {
  final String id;
  final String nome;
  final bool serial;
  Scanner(this.id, this.nome, this.serial);

  String get descricao =>
      serial ? 'Serial $id' : (nome.isEmpty ? 'Scanner USB' : 'USB · $nome');

  Map<String, dynamic> toJson() => {'id': id, 'nome': nome, 'serial': serial};
  Scanner.fromJson(Map<String, dynamic> j)
    : id = j['id'] ?? '',
      nome = j['nome'] ?? '',
      serial = j['serial'] ?? false;
}

class Leitura {
  final DateTime hora = DateTime.now();
  final String id;
  final String nome;
  final bool serial;
  final String texto;
  Leitura(this.id, this.nome, this.serial, this.texto);
}

/// Tecla informada pelo agente: só de qual aparelho veio.
class Origem {
  final DateTime hora = DateTime.now();
  final String id;
  final String nome;
  Origem(this.id, this.nome);
}

/// Tecla recebida pelo app (já com o caractere, convertido pelo sistema).
class TeclaApp {
  final DateTime hora = DateTime.now();
  final KeyDownEvent evento;
  TeclaApp(this.evento);
}

/// Texto sendo digitado agora por um aparelho USB.
class Digitando {
  final String id;
  final String nome;
  String texto = '';
  Digitando(this.id, this.nome);
}

class Tela extends StatefulWidget {
  const Tela({super.key});

  @override
  State<Tela> createState() => _TelaState();
}

class _TelaState extends State<Tela> {
  final _agente = Agente();
  SharedPreferences? _prefs;

  final List<Scanner> _scanners = [];
  bool _identificando = false;
  String _status = 'Iniciando...';
  String _aviso = '';
  bool _conectado = false;

  final List<Origem> _origens = [];
  final List<TeclaApp> _teclas = [];
  final Map<String, Digitando> _digitando = {};
  final List<Leitura> _leituras = [];

  @override
  void initState() {
    super.initState();
    HardwareKeyboard.instance.addHandler(_teclaApp);
    _iniciar();
  }

  @override
  void dispose() {
    HardwareKeyboard.instance.removeHandler(_teclaApp);
    _agente.parar();
    super.dispose();
  }

  Future<void> _iniciar() async {
    _prefs = await SharedPreferences.getInstance();
    final salvos = _prefs!.getStringList('scanners') ?? [];
    setState(() {
      for (final s in salvos) {
        try {
          _scanners.add(Scanner.fromJson(jsonDecode(s)));
        } catch (_) {}
      }
    });

    _agente.eventos.listen(_receber);
    _agente.saiu.then((codigo) {
      if (mounted) {
        setState(() {
          _conectado = false;
          _status = 'O agente parou (código $codigo)';
        });
      }
    });
    try {
      await _agente.iniciar();
      _atualizarPortas();
    } catch (e) {
      setState(() => _status = e.toString());
    }
  }

  void _salvarScanners() {
    _prefs?.setStringList(
      'scanners',
      _scanners.map((s) => jsonEncode(s.toJson())).toList(),
    );
  }

  /// Diz ao agente quais portas seriais abrir: todas durante o cadastro;
  /// fora dele, só as dos scanners seriais cadastrados.
  void _atualizarPortas() {
    _agente.enviar(
      _identificando
          ? {'cmd': 'portas', 'todas': true}
          : {
              'cmd': 'portas',
              'lista': [
                for (final s in _scanners)
                  if (s.serial) s.id,
              ],
            },
    );
  }

  void _cadastrar(bool ligar) {
    setState(() {
      _identificando = ligar;
      if (ligar) _aviso = '';
    });
    _atualizarPortas();
  }

  /// Posição do scanner cadastrado (0 ou 1), ou -1 se não for scanner.
  int _scanner(String id) => _scanners.indexWhere((s) => s.id == id);

  void _receber(EventoAgente e) {
    setState(() {
      switch (e.tipo) {
        case 'pronto':
          _conectado = true;
          _status = 'Escutando teclados, scanners USB e portas seriais';
        case 'erro':
          _status = e.texto;
        case 'aviso':
          _aviso = e.texto;
        case 'serial':
          _novaLeitura(Leitura(e.id, e.nome, true, e.texto));
        case 'tecla':
          _origens.add(Origem(e.id, e.nome));
          _parear();
      }
    });
  }

  /// Toda tecla apertada com o app em foco (não consome a tecla).
  bool _teclaApp(KeyEvent e) {
    if (e is KeyDownEvent && !e.synthesized) {
      setState(() {
        _teclas.add(TeclaApp(e));
        _parear();
      });
    }
    return false;
  }

  /// O caractere vem do app; a origem vem do agente. Os dois chegam por
  /// caminhos diferentes, na mesma ordem: a 1ª tecla de um casa com a 1ª do
  /// outro. Sobras com mais de 1 segundo (ex: tecla com o app sem foco) são
  /// descartadas.
  void _parear() {
    final limite = DateTime.now().subtract(const Duration(seconds: 1));
    _origens.removeWhere((o) => o.hora.isBefore(limite));
    _teclas.removeWhere((t) => t.hora.isBefore(limite));
    while (_origens.isNotEmpty && _teclas.isNotEmpty) {
      _aplicar(_origens.removeAt(0), _teclas.removeAt(0).evento);
    }
  }

  void _aplicar(Origem o, KeyDownEvent e) {
    final d = _digitando.putIfAbsent(o.id, () => Digitando(o.id, o.nome));
    final tecla = e.logicalKey;
    if (tecla == LogicalKeyboardKey.enter ||
        tecla == LogicalKeyboardKey.numpadEnter) {
      if (d.texto.isNotEmpty) {
        _novaLeitura(Leitura(d.id, d.nome, false, d.texto));
      }
      _digitando.remove(o.id);
    } else if (tecla == LogicalKeyboardKey.backspace) {
      if (d.texto.isNotEmpty) {
        d.texto = d.texto.characters.skipLast(1).toString();
      }
    } else {
      final c = e.character;
      if (c != null && c.isNotEmpty && c.codeUnitAt(0) >= 0x20) {
        d.texto += c;
      }
    }
  }

  /// Leitura completa (Enter no USB, fim de linha na serial). Se está
  /// aguardando o cadastro, a origem desta leitura vira um scanner.
  void _novaLeitura(Leitura l) {
    if (_identificando &&
        _scanner(l.id) < 0 &&
        _scanners.length < maxScanners) {
      _scanners.add(Scanner(l.id, l.nome, l.serial));
      _salvarScanners();
      _identificando = false;
      _atualizarPortas();
    }
    _leituras.insert(0, l);
    if (_leituras.length > 200) _leituras.removeLast();
  }

  void _remover(int i) {
    setState(() {
      _scanners.removeAt(i);
      _salvarScanners();
    });
    _atualizarPortas();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Teclado x Scanner')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _cartaoScanners(),
            const SizedBox(height: 8),
            Row(
              children: [
                Icon(
                  Icons.circle,
                  size: 10,
                  color: _conectado ? Colors.green : Colors.red,
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(_status, style: const TextStyle(fontSize: 12)),
                ),
              ],
            ),
            if (_aviso.isNotEmpty)
              Text(
                _aviso,
                style: const TextStyle(fontSize: 11, color: Colors.grey),
              ),
            const Divider(height: 24),
            const Text(
              'Digitando agora (com o app em foco):',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 4),
            if (_digitando.isEmpty)
              const Text('—', style: TextStyle(color: Colors.grey))
            else
              ..._digitando.values.map(
                (d) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  child: Row(
                    children: [
                      _etiqueta(d.id, false),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          '${d.texto}▌',
                          style: const TextStyle(
                            fontFamily: 'monospace',
                            fontSize: 16,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            const Divider(height: 24),
            Row(
              children: [
                const Expanded(
                  child: Text(
                    'Leituras:',
                    style: TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                TextButton(
                  onPressed: () => setState(_leituras.clear),
                  child: const Text('Limpar lista'),
                ),
              ],
            ),
            Expanded(
              child: ListView.builder(
                itemCount: _leituras.length,
                itemBuilder: (_, i) => _linhaLeitura(_leituras[i]),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _cartaoScanners() {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                const Icon(Icons.qr_code_scanner),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    _identificando
                        ? 'Faça uma leitura no scanner...'
                        : 'Scanners (${_scanners.length}/$maxScanners)',
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                if (_identificando)
                  OutlinedButton(
                    onPressed: () => _cadastrar(false),
                    child: const Text('Cancelar'),
                  )
                else
                  FilledButton(
                    onPressed: _scanners.length >= maxScanners
                        ? null
                        : () => _cadastrar(true),
                    child: const Text('Adicionar scanner'),
                  ),
              ],
            ),
            if (_identificando)
              const Padding(
                padding: EdgeInsets.only(top: 4),
                child: Text(
                  'A próxima leitura define o scanner (USB ou serial). '
                  'Para USB, deixe o app em foco. As portas seriais ficam '
                  'abertas só durante o cadastro.',
                  style: TextStyle(fontSize: 12),
                ),
              ),
            if (_scanners.isEmpty && !_identificando)
              const Padding(
                padding: EdgeInsets.only(top: 4),
                child: Text(
                  'Nenhum scanner cadastrado: tudo aparece como TECLADO.',
                  style: TextStyle(fontSize: 12),
                ),
              ),
            for (var i = 0; i < _scanners.length; i++)
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Row(
                  children: [
                    _etiqueta(_scanners[i].id, _scanners[i].serial),
                    const SizedBox(width: 8),
                    Expanded(child: Text(_scanners[i].descricao)),
                    IconButton(
                      tooltip: 'Remover',
                      icon: const Icon(Icons.close, size: 18),
                      onPressed: () => _remover(i),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _linhaLeitura(Leitura l) {
    String dois(int n) => n.toString().padLeft(2, '0');
    final hora =
        '${dois(l.hora.hour)}:${dois(l.hora.minute)}:${dois(l.hora.second)}';
    final detalhe = l.serial ? l.id : l.nome;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            hora,
            style: const TextStyle(fontFamily: 'monospace', color: Colors.grey),
          ),
          const SizedBox(width: 8),
          _etiqueta(l.id, l.serial),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SelectableText(
                  l.texto,
                  style: const TextStyle(fontFamily: 'monospace', fontSize: 16),
                ),
                if (detalhe.isNotEmpty)
                  Text(
                    detalhe,
                    style: const TextStyle(fontSize: 11, color: Colors.grey),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  /// SCANNER 1 / SCANNER 2 para os cadastrados; o resto é TECLADO (USB) ou
  /// SERIAL (porta serial não cadastrada).
  Widget _etiqueta(String id, bool serial) {
    final i = _scanner(id);
    final Color cor;
    final String texto;
    if (i >= 0) {
      cor = coresScanner[i];
      texto = 'SCANNER ${i + 1}';
    } else if (serial) {
      cor = corSerial;
      texto = 'SERIAL';
    } else {
      cor = corTeclado;
      texto = 'TECLADO';
    }
    return Container(
      width: 80,
      padding: const EdgeInsets.symmetric(vertical: 2),
      decoration: BoxDecoration(
        color: cor,
        borderRadius: BorderRadius.circular(4),
      ),
      alignment: Alignment.center,
      child: Text(
        texto,
        style: const TextStyle(
          color: Colors.white,
          fontSize: 11,
          fontWeight: FontWeight.bold,
        ),
      ),
    );
  }
}
