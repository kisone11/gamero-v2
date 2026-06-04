export class ApiConfig {
  // DevEco Emulator can usually reach the host machine through 10.0.2.2.
  // If you run on a real phone, replace this with your PC LAN IP, for example:
  // http://192.168.1.10:8081/api/v1
  static readonly baseUrl: string = 'http://10.0.2.2:8081/api/v1';

  static readonly adminEmail: string = 'admin@gamero.local';
  static readonly adminPassword: string = 'Gamero123';
}
