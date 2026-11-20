import { readSharedPreference, writeSharedPreference } from './locale-preference';

export type Locale = 'en' | 'pt' | 'es';
export type Preference = Locale | 'system';
const key = 'plunz.lounge-site.locale';

export function detectLocale(tags: readonly string[] = typeof navigator === 'undefined' ? [] : navigator.languages): Locale {
  const preferred = tags.length ? tags : typeof navigator === 'undefined' ? [] : [navigator.language];
  for (const tag of preferred) {
    const base = tag.trim().toLowerCase().split(/[-_]/)[0];
    if (base === 'en' || base === 'pt' || base === 'es') return base;
  }
  return 'en';
}

export function readPreference(): Preference {
  try {
    return readSharedPreference(localStorage, key);
  } catch { return 'system'; }
}

export function savePreference(value: Preference): void {
  try {
    writeSharedPreference(localStorage, key, value);
  } catch { /* Private browsing can disable storage. */ }
}

const en = {
  skip: 'Skip to content', language: 'Language', system: 'System language',
  navIdea: 'The idea', navFeatures: 'Features', navInstall: 'Download',
  title: 'Lounge — Your addons on the big screen',
  description: 'Lounge is a Linux TV app for Stremio addons, made for a remote control. Download it for Linux.',
  eyebrow: 'LINUX TV APP',
  headlineStart: 'Your addons,', headlineEnd: 'on the big screen.',
  lead: 'Lounge turns a Linux PC connected to a TV into a living-room player for Stremio addons. Browse catalogs, pick a source, and watch with a remote, a gamepad, or a keyboard.',
  download: 'Download for Linux', downloadMeta: 'x86_64 · tar.gz · latest release', source: 'Source on GitHub',
  mockHome: 'Home', mockMovies: 'Movies', mockSeries: 'Series', mockSearch: 'Search', mockKicker: 'Continue watching', mockMeta: 'Drama · 2h 04m · 38 min left',
  ideaEyebrow: 'THE IDEA',
  ideaTitle: 'A couch-first player for the addons you already use.',
  ideaBody: 'Stremio addons describe catalogs, titles, and streams. Lounge reads them directly and wraps them in a 10-foot interface built for D-pad remotes, with the video playing in the same window. No browser tabs, no mouse, no extra services.',
  pillars: [
    { title: 'Made for remotes', body: 'Focus moves the way the arrow keys expect, from 720p to 4K. Any remote that sends keyboard keys works as it is.' },
    { title: 'Plays in place', body: 'libmpv draws the video under the interface, with audio tracks, subtitles, speed, and source switching on top.' },
    { title: 'Nothing else to run', body: 'Direct streams and BitTorrent play through an embedded engine, so Stremio Service is not needed. Torrent data is deleted when playback stops.' },
  ],
  featuresEyebrow: 'FEATURES', featuresTitle: 'Everything for a night in.',
  features: [
    'Install addons by URL. Cinemeta and OpenSubtitles v3 come installed.',
    'Optional Stremio sign-in with a QR code or email, mirroring your addons read-only.',
    'Home, Movies, and Series rows, search, and genre filters.',
    'Sources grouped by addon, with quality and size.',
    'Continue watching, resume, and the next episode played automatically.',
    'Follows your GTK theme, light or dark, with its accent color.',
    'English, Portuguese, and Spanish.',
  ],
  controlsTitle: 'Controls',
  controls: [
    ['Arrows', 'Move focus; seek 10 s while watching'],
    ['Enter / OK', 'Select; pause and show the controls'],
    ['Back / Escape', 'Go back'],
    ['Space', 'Play or pause'],
    ['F11', 'Toggle fullscreen'],
  ],
  installEyebrow: 'DOWNLOAD', installTitle: 'Install Lounge.',
  installLead: 'Lounge is built for Arch Linux and Omarchy on x86_64. Other distributions work when they provide GTK 4, WebKitGTK 6.0, and libmpv.',
  steps: [
    { title: 'Install the libraries it uses', code: 'sudo pacman -S --needed gtk4 webkitgtk-6.0 mpv libepoxy' },
    { title: 'Download and extract the archive', code: 'tar -xzf lounge-linux-x86_64.tar.gz' },
    { title: 'Run the installer', code: 'cd lounge-*/ && ./install.sh' },
  ],
  installNote: 'The installer puts lounge in ~/.local/bin, with a desktop entry and an icon, for your user only. Lounge starts fullscreen; set LOUNGE_WINDOWED=1 to start it in a window.',
  copy: 'Copy', copied: 'Copied',
  checksum: 'SHA-256 checksum', releases: 'All releases', fromSource: 'Build from source',
  pairEyebrow: 'WORKS WELL WITH', pairBody: 'Nexus, a TV panel that opens Lounge and controls volume, brightness, and a sleep timer from the same remote.', pairLink: 'Meet Nexus',
  disclaimer: 'Lounge is an independent app and is not affiliated with Stremio. It does not host or provide content: you choose the addons you install and are responsible for how you use them.',
  partOf: 'A PLUNZ app', privacy: 'Privacy', terms: 'Terms',
};

export type Copy = typeof en;

export const copy: Record<Locale, Copy> = {
  en,
  pt: {
    skip: 'Pular para o conteúdo', language: 'Idioma', system: 'Idioma do sistema',
    navIdea: 'A ideia', navFeatures: 'Recursos', navInstall: 'Download',
    title: 'Lounge — Seus addons na tela grande',
    description: 'Lounge é um app de TV para Linux que usa addons do Stremio, feito para controle remoto. Baixe para Linux.',
    eyebrow: 'APP DE TV PARA LINUX',
    headlineStart: 'Seus addons,', headlineEnd: 'na tela grande.',
    lead: 'O Lounge transforma um PC com Linux ligado à TV em um player de sala para addons do Stremio. Navegue pelos catálogos, escolha uma fonte e assista com controle remoto, gamepad ou teclado.',
    download: 'Baixar para Linux', downloadMeta: 'x86_64 · tar.gz · última versão', source: 'Código no GitHub',
    mockHome: 'Início', mockMovies: 'Filmes', mockSeries: 'Séries', mockSearch: 'Buscar', mockKicker: 'Continuar assistindo', mockMeta: 'Drama · 2h 04min · faltam 38 min',
    ideaEyebrow: 'A IDEIA',
    ideaTitle: 'Um player feito para o sofá, com os addons que você já usa.',
    ideaBody: 'Os addons do Stremio descrevem catálogos, títulos e streams. O Lounge lê esses addons diretamente e os apresenta em uma interface para ver de longe, feita para controles com setas, com o vídeo tocando na mesma janela. Sem abas de navegador, sem mouse, sem serviços extras.',
    pillars: [
      { title: 'Feito para o controle', body: 'O foco anda como as setas sugerem, de 720p a 4K. Qualquer controle que envie teclas de teclado funciona como está.' },
      { title: 'Toca no lugar', body: 'A libmpv desenha o vídeo sob a interface, com faixas de áudio, legendas, velocidade e troca de fonte por cima.' },
      { title: 'Nada mais para rodar', body: 'Streams diretos e BitTorrent tocam por um mecanismo embutido, sem precisar do Stremio Service. Os dados do torrent são apagados quando a reprodução para.' },
    ],
    featuresEyebrow: 'RECURSOS', featuresTitle: 'Tudo para uma noite em casa.',
    features: [
      'Instale addons por URL. Cinemeta e OpenSubtitles v3 já vêm instalados.',
      'Login opcional no Stremio por QR code ou e-mail, espelhando seus addons somente para leitura.',
      'Fileiras de Início, Filmes e Séries, busca e filtros por gênero.',
      'Fontes agrupadas por addon, com qualidade e tamanho.',
      'Continuar assistindo, retomar e próximo episódio automático.',
      'Segue o tema GTK, claro ou escuro, com a cor de destaque dele.',
      'Inglês, português e espanhol.',
    ],
    controlsTitle: 'Controles',
    controls: [
      ['Setas', 'Mover o foco; avançar ou voltar 10 s no vídeo'],
      ['Enter / OK', 'Selecionar; pausar e mostrar os controles'],
      ['Voltar / Esc', 'Voltar'],
      ['Espaço', 'Tocar ou pausar'],
      ['F11', 'Alternar tela cheia'],
    ],
    installEyebrow: 'DOWNLOAD', installTitle: 'Instale o Lounge.',
    installLead: 'O Lounge é feito para Arch Linux e Omarchy em x86_64. Outras distribuições funcionam se tiverem GTK 4, WebKitGTK 6.0 e libmpv.',
    steps: [
      { title: 'Instale as bibliotecas que ele usa', code: 'sudo pacman -S --needed gtk4 webkitgtk-6.0 mpv libepoxy' },
      { title: 'Baixe e extraia o arquivo', code: 'tar -xzf lounge-linux-x86_64.tar.gz' },
      { title: 'Rode o instalador', code: 'cd lounge-*/ && ./install.sh' },
    ],
    installNote: 'O instalador coloca o lounge em ~/.local/bin, com atalho e ícone, só para o seu usuário. O Lounge abre em tela cheia; defina LOUNGE_WINDOWED=1 para abrir em janela.',
    copy: 'Copiar', copied: 'Copiado',
    checksum: 'Checksum SHA-256', releases: 'Todas as versões', fromSource: 'Compilar do código',
    pairEyebrow: 'COMBINA COM', pairBody: 'Nexus, um painel de TV que abre o Lounge e controla volume, brilho e timer de desligamento com o mesmo controle.', pairLink: 'Conheça o Nexus',
    disclaimer: 'O Lounge é um app independente, sem vínculo com o Stremio. Ele não hospeda nem fornece conteúdo: você escolhe os addons que instala e é responsável pelo uso que faz deles.',
    partOf: 'Um app PLUNZ', privacy: 'Privacidade', terms: 'Termos',
  },
  es: {
    skip: 'Ir al contenido', language: 'Idioma', system: 'Idioma del sistema',
    navIdea: 'La idea', navFeatures: 'Funciones', navInstall: 'Descarga',
    title: 'Lounge — Tus addons en la pantalla grande',
    description: 'Lounge es una app de TV para Linux que usa addons de Stremio, hecha para control remoto. Descárgala para Linux.',
    eyebrow: 'APP DE TV PARA LINUX',
    headlineStart: 'Tus addons,', headlineEnd: 'en la pantalla grande.',
    lead: 'Lounge convierte un PC con Linux conectado a la TV en un reproductor de salón para addons de Stremio. Explora catálogos, elige una fuente y mira con un control remoto, un mando o un teclado.',
    download: 'Descargar para Linux', downloadMeta: 'x86_64 · tar.gz · última versión', source: 'Código en GitHub',
    mockHome: 'Inicio', mockMovies: 'Películas', mockSeries: 'Series', mockSearch: 'Buscar', mockKicker: 'Seguir viendo', mockMeta: 'Drama · 2 h 04 min · quedan 38 min',
    ideaEyebrow: 'LA IDEA',
    ideaTitle: 'Un reproductor pensado para el sofá, con los addons que ya usas.',
    ideaBody: 'Los addons de Stremio describen catálogos, títulos y streams. Lounge los lee directamente y los presenta en una interfaz para ver a distancia, hecha para controles con flechas, con el video en la misma ventana. Sin pestañas de navegador, sin ratón, sin servicios extra.',
    pillars: [
      { title: 'Hecho para el control', body: 'El foco se mueve como indican las flechas, de 720p a 4K. Cualquier control que envíe teclas de teclado funciona tal cual.' },
      { title: 'Reproduce en su sitio', body: 'libmpv dibuja el video bajo la interfaz, con pistas de audio, subtítulos, velocidad y cambio de fuente encima.' },
      { title: 'Nada más que ejecutar', body: 'Los streams directos y BitTorrent se reproducen con un motor integrado, sin necesidad de Stremio Service. Los datos del torrent se borran cuando termina la reproducción.' },
    ],
    featuresEyebrow: 'FUNCIONES', featuresTitle: 'Todo para una noche en casa.',
    features: [
      'Instala addons por URL. Cinemeta y OpenSubtitles v3 vienen instalados.',
      'Inicio de sesión opcional en Stremio con código QR o correo, que refleja tus addons en solo lectura.',
      'Filas de Inicio, Películas y Series, búsqueda y filtros por género.',
      'Fuentes agrupadas por addon, con calidad y tamaño.',
      'Seguir viendo, reanudar y siguiente episodio automático.',
      'Sigue tu tema GTK, claro u oscuro, con su color de acento.',
      'Inglés, portugués y español.',
    ],
    controlsTitle: 'Controles',
    controls: [
      ['Flechas', 'Mover el foco; avanzar o retroceder 10 s en el video'],
      ['Enter / OK', 'Seleccionar; pausar y mostrar los controles'],
      ['Atrás / Esc', 'Volver'],
      ['Espacio', 'Reproducir o pausar'],
      ['F11', 'Alternar pantalla completa'],
    ],
    installEyebrow: 'DESCARGA', installTitle: 'Instala Lounge.',
    installLead: 'Lounge está hecho para Arch Linux y Omarchy en x86_64. Otras distribuciones funcionan si ofrecen GTK 4, WebKitGTK 6.0 y libmpv.',
    steps: [
      { title: 'Instala las bibliotecas que usa', code: 'sudo pacman -S --needed gtk4 webkitgtk-6.0 mpv libepoxy' },
      { title: 'Descarga y extrae el archivo', code: 'tar -xzf lounge-linux-x86_64.tar.gz' },
      { title: 'Ejecuta el instalador', code: 'cd lounge-*/ && ./install.sh' },
    ],
    installNote: 'El instalador coloca lounge en ~/.local/bin, con un acceso directo y un icono, solo para tu usuario. Lounge abre a pantalla completa; define LOUNGE_WINDOWED=1 para abrirlo en una ventana.',
    copy: 'Copiar', copied: 'Copiado',
    checksum: 'Suma SHA-256', releases: 'Todas las versiones', fromSource: 'Compilar desde el código',
    pairEyebrow: 'VA BIEN CON', pairBody: 'Nexus, un panel de TV que abre Lounge y controla el volumen, el brillo y un temporizador de apagado con el mismo control.', pairLink: 'Conoce Nexus',
    disclaimer: 'Lounge es una app independiente y no está afiliada a Stremio. No aloja ni proporciona contenido: tú eliges los addons que instalas y eres responsable del uso que haces de ellos.',
    partOf: 'Una app de PLUNZ', privacy: 'Privacidad', terms: 'Términos',
  },
};
