export interface Cookie {
  name: string;
  value: string;
  domain: string;
  path: string;
  expires?: number;
  secure: boolean;
  httpOnly: boolean;
  sameSite?: string;
}

export interface SessionStatus {
  valid: boolean;
  message: string;
  exists: boolean;
  host?: string;
  flavor?: string;
}

export interface ExtensionSettings {
  backend_url: string;
}
