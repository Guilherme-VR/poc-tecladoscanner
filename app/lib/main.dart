import 'package:flutter/material.dart';
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

const corTeclado = Color(0xFF1E66D0);
const corScanner = Color(0xFFE07B00);

class Leitura {
  final DateTime hora;
  final bool teclado;
  final String texto;
  final String id;
  final String nome;
  Leitura(this.hora, this.teclado, this.texto, this.id, this.nome);
}

/// Texto sendo digitado agora por uma unidade física.
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

  String? _tecladoId;
  String _tecladoNome = '';
  bool _identificando = false;
  String _status = 'Iniciando...';
  bool _conectado = false;

  final Map<String, Digitando> _digitando = {};
  final List<Leitura> _leituras = [];

  @override
  void initState() {
    super.initState();
    _iniciar();
  }

  @override
  void dispose() {
    _agente.parar();
    super.dispose();
  }

  Future<void> _iniciar() async {
    _prefs = await SharedPreferences.getInstance();
    setState(() {
      _tecladoId = _prefs!.getString('tecladoId');
      _tecladoNome = _prefs!.getString('tecladoNome') ?? '';
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
    } catch (e) {
      setState(() => _status = e.toString());
    }
  }

  bool _ehTeclado(String id) => _tecladoId != null && id == _tecladoId;

  void _receber(EventoAgente e) {
    setState(() {
      switch (e.tipo) {
        case 'pronto':
          _conectado = true;
          _status = 'Escutando os teclados';
        case 'erro':
          _status = e.texto;
        case 'tecla' || 'enter' || 'apagar':
          if (_identificando) {
            _salvarTeclado(e.id, e.nome);
            return;
          }
          final d = _digitando.putIfAbsent(e.disp, () => Digitando(e.id, e.nome));
          if (e.tipo == 'tecla') {
            d.texto += e.texto;
          } else if (e.tipo == 'apagar') {
            if (d.texto.isNotEmpty) {
              d.texto = d.texto.characters.skipLast(1).toString();
            }
          } else {
            if (d.texto.isNotEmpty) {
              _leituras.insert(
                  0, Leitura(DateTime.now(), _ehTeclado(d.id), d.texto, d.id, d.nome));
            }
            _digitando.remove(e.disp);
          }
      }
    });
  }

  void _salvarTeclado(String id, String nome) {
    _identificando = false;
    _tecladoId = id;
    _tecladoNome = nome;
    _prefs?.setString('tecladoId', id);
    _prefs?.setString('tecladoNome', nome);
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
            _cartaoTeclado(),
            const SizedBox(height: 8),
            Row(children: [
              Icon(Icons.circle, size: 10, color: _conectado ? Colors.green : Colors.red),
              const SizedBox(width: 6),
              Expanded(child: Text(_status, style: const TextStyle(fontSize: 12))),
            ]),
            const Divider(height: 24),
            const Text('Digitando agora:', style: TextStyle(fontWeight: FontWeight.bold)),
            const SizedBox(height: 4),
            if (_digitando.isEmpty)
              const Text('—', style: TextStyle(color: Colors.grey))
            else
              ..._digitando.values.map((d) => Padding(
                    padding: const EdgeInsets.symmetric(vertical: 2),
                    child: Row(children: [
                      _etiqueta(_ehTeclado(d.id)),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text('${d.texto}▌',
                            style: const TextStyle(fontFamily: 'monospace', fontSize: 16)),
                      ),
                    ]),
                  )),
            const Divider(height: 24),
            Row(children: [
              const Expanded(
                  child: Text('Leituras:', style: TextStyle(fontWeight: FontWeight.bold))),
              TextButton(
                onPressed: () => setState(_leituras.clear),
                child: const Text('Limpar lista'),
              ),
            ]),
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

  Widget _cartaoTeclado() {
    final String titulo;
    final String detalhe;
    if (_identificando) {
      titulo = 'Aperte qualquer tecla no teclado...';
      detalhe = 'A próxima tecla define qual aparelho é o teclado';
    } else if (_tecladoId == null) {
      titulo = 'Teclado não identificado';
      detalhe = 'Tudo está sendo tratado como SCANNER. Clique em "Identificar teclado".';
    } else {
      titulo = _tecladoNome.isEmpty ? 'Teclado' : _tecladoNome;
      detalhe = 'ID: $_tecladoId';
    }
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(children: [
          const Icon(Icons.keyboard, color: corTeclado),
          const SizedBox(width: 12),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(titulo, style: const TextStyle(fontWeight: FontWeight.bold)),
              Text(detalhe, style: const TextStyle(fontSize: 12)),
            ]),
          ),
          _identificando
              ? OutlinedButton(
                  onPressed: () => setState(() => _identificando = false),
                  child: const Text('Cancelar'))
              : FilledButton(
                  onPressed: () => setState(() => _identificando = true),
                  child: const Text('Identificar teclado')),
        ]),
      ),
    );
  }

  Widget _linhaLeitura(Leitura l) {
    String dois(int n) => n.toString().padLeft(2, '0');
    final hora = '${dois(l.hora.hour)}:${dois(l.hora.minute)}:${dois(l.hora.second)}';
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(hora, style: const TextStyle(fontFamily: 'monospace', color: Colors.grey)),
        const SizedBox(width: 8),
        _etiqueta(l.teclado),
        const SizedBox(width: 8),
        Expanded(
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            SelectableText(l.texto,
                style: const TextStyle(fontFamily: 'monospace', fontSize: 16)),
            if (!l.teclado)
              Text('${l.nome} ${l.id}'.trim(),
                  style: const TextStyle(fontSize: 11, color: Colors.grey)),
          ]),
        ),
      ]),
    );
  }

  Widget _etiqueta(bool teclado) {
    final cor = teclado ? corTeclado : corScanner;
    return Container(
      width: 72,
      padding: const EdgeInsets.symmetric(vertical: 2),
      decoration: BoxDecoration(color: cor, borderRadius: BorderRadius.circular(4)),
      alignment: Alignment.center,
      child: Text(teclado ? 'TECLADO' : 'SCANNER',
          style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.bold)),
    );
  }
}
