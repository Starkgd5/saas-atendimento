class FormatterService {
  private static instance: FormatterService;

  private constructor() {}

  static getInstance(): FormatterService {
    if (!FormatterService.instance) {
      FormatterService.instance = new FormatterService();
    }
    return FormatterService.instance;
  }

  // Moeda
  formatCurrency(value: number): string {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: 'BRL',
    }).format(value);
  }

  // Data
  formatDate(date: string | Date): string {
    const d = typeof date === 'string' ? new Date(date) : date;
    return new Intl.DateTimeFormat('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    }).format(d);
  }

  formatDateTime(date: string | Date): string {
    const d = typeof date === 'string' ? new Date(date) : date;
    return new Intl.DateTimeFormat('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(d);
  }

  formatTime(date: string | Date): string {
    const d = typeof date === 'string' ? new Date(date) : date;
    return new Intl.DateTimeFormat('pt-BR', {
      hour: '2-digit',
      minute: '2-digit',
    }).format(d);
  }

  // Número
  formatNumber(value: number, decimals: number = 0): string {
    return new Intl.NumberFormat('pt-BR', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    }).format(value);
  }

  // Percentual
  formatPercent(value: number): string {
    return new Intl.NumberFormat('pt-BR', {
      style: 'percent',
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    }).format(value / 100);
  }

  // Telefone
  formatPhone(phone: string): string {
    const cleaned = phone.replace(/\D/g, '');
    if (cleaned.length === 10) {
      return `(${cleaned.slice(0, 2)}) ${cleaned.slice(2, 6)}-${cleaned.slice(6, 10)}`;
    }
    if (cleaned.length === 11) {
      return `(${cleaned.slice(0, 2)}) ${cleaned.slice(2, 7)}-${cleaned.slice(7, 11)}`;
    }
    return phone;
  }

  // CPF/CNPJ
  formatCpf(cpf: string): string {
    const cleaned = cpf.replace(/\D/g, '');
    if (cleaned.length === 11) {
      return cleaned.replace(/(\d{3})(\d{3})(\d{3})(\d{2})/, '$1.$2.$3-$4');
    }
    return cpf;
  }

  formatCnpj(cnpj: string): string {
    const cleaned = cnpj.replace(/\D/g, '');
    if (cleaned.length === 14) {
      return cleaned.replace(/(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})/, '$1.$2.$3/$4-$5');
    }
    return cnpj;
  }

  // Truncar texto
  truncate(text: string, length: number = 50, suffix: string = '...'): string {
    if (text.length <= length) return text;
    return text.slice(0, length) + suffix;
  }

  // Capitalizar
  capitalize(text: string): string {
    return text.charAt(0).toUpperCase() + text.slice(1).toLowerCase();
  }

  // Slug
  slugify(text: string): string {
    return text
      .normalize('NFKD')
      .replace(/[\u0300-\u036f]/g, '')
      .toLowerCase()
      .trim()
      .replace(/\s+/g, '-')
      .replace(/[^\w-]+/g, '')
      .replace(/--+/g, '-');
  }

  // Tempo relativo
  timeAgo(date: string | Date): string {
    const now = new Date();
    const past = typeof date === 'string' ? new Date(date) : date;
    const diff = now.getTime() - past.getTime();

    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    const months = Math.floor(days / 30);
    const years = Math.floor(months / 12);

    if (years > 0) return `há ${years} ${years === 1 ? 'ano' : 'anos'}`;
    if (months > 0) return `há ${months} ${months === 1 ? 'mês' : 'meses'}`;
    if (days > 0) return `há ${days} ${days === 1 ? 'dia' : 'dias'}`;
    if (hours > 0) return `há ${hours} ${hours === 1 ? 'hora' : 'horas'}`;
    if (minutes > 0) return `há ${minutes} ${minutes === 1 ? 'minuto' : 'minutos'}`;
    return 'agora mesmo';
  }
}

export default FormatterService.getInstance();