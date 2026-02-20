#include <gpiod.h>
/*
  Comple：make
  Run: ./bme280
  
  This Demo is tested on Raspberry PI 3B+
  you can use I2C or SPI interface to test this Demo
  When you use I2C interface,the default Address in this demo is 0X77
  When you use SPI interface,PIN 27 define SPI_CS
*/
#include "bme280.h"
#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/ioctl.h>
#include <linux/spi/spidev.h>
#include <stdint.h>
#include <string.h>
#include <stdlib.h>

//Raspberry 3B+ platform's default SPI channel
#define channel 0  

//Default write it to the register in one time
#define USESPISINGLEREADWRITE 0 

//This definition you use I2C or SPI to drive the bme280
//When it is 1 means use I2C interface, When it is 0,use SPI interface
#define USEIIC 0


#if(USEIIC)
#include <string.h>
#include <stdlib.h>
#include <linux/i2c-dev.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <fcntl.h>
//Raspberry 3B+ platform's default I2C device file
#define IIC_Dev  "/dev/i2c-1"
	
int fd;

void user_delay_ms(uint32_t period)
{
  usleep(period*1000);
}

int8_t user_i2c_read(uint8_t id, uint8_t reg_addr, uint8_t *data, uint16_t len)
{
  write(fd, &reg_addr,1);
  read(fd, data, len);
  return 0;
}

int8_t user_i2c_write(uint8_t id, uint8_t reg_addr, uint8_t *data, uint16_t len)
{
  int8_t *buf;
  buf = malloc(len +1);
  buf[0] = reg_addr;
  memcpy(buf +1, data, len);
  write(fd, buf, len +1);
  free(buf);
  return 0;
}
#else


#define CS_GPIO 27
#define SPI_DEV "/dev/spidev0.0"
static int spi_fd = -1;
static struct gpiod_chip *chip = NULL;
static struct gpiod_line *cs_line = NULL;

void SPI_BME280_CS_High(void)
{
  if (cs_line) {
    gpiod_line_set_value(cs_line, 1);
  }
}

void SPI_BME280_CS_Low(void)
{
  if (cs_line) {
    gpiod_line_set_value(cs_line, 0);
  }
}

void user_delay_ms(uint32_t period)
{
  usleep(period*1000);
}

int8_t user_spi_read(uint8_t dev_id, uint8_t reg_addr, uint8_t *reg_data, uint16_t len)
{
  int8_t rslt = 0;
  struct spi_ioc_transfer tr[2];
  uint8_t tx[1];
  tx[0] = reg_addr | 0x80; // set MSB for read
  memset(tr, 0, sizeof(tr));

  SPI_BME280_CS_Low();

  tr[0].tx_buf = (unsigned long)tx;
  tr[0].rx_buf = 0;
  tr[0].len = 1;
  tr[0].speed_hz = 2000000;
  tr[0].bits_per_word = 8;

  tr[1].tx_buf = 0;
  tr[1].rx_buf = (unsigned long)reg_data;
  tr[1].len = len;
  tr[1].speed_hz = 2000000;
  tr[1].bits_per_word = 8;

  if (ioctl(spi_fd, SPI_IOC_MESSAGE(2), tr) < 1) {
    rslt = -1;
  }

  SPI_BME280_CS_High();
  return rslt;
}

int8_t user_spi_write(uint8_t dev_id, uint8_t reg_addr, uint8_t *reg_data, uint16_t len)
{
    int8_t rslt = 0;
    uint8_t *tx = malloc(len + 1);
    if (!tx) return -1;
    tx[0] = reg_addr & 0x7F; // clear MSB for write
    memcpy(&tx[1], reg_data, len);

    SPI_BME280_CS_Low();
    if (write(spi_fd, tx, len + 1) != len + 1) {
        rslt = -1;
    }
    SPI_BME280_CS_High();
    free(tx);
    return rslt;
}
#endif

void print_sensor_data(struct bme280_data *comp_data)
{
#ifdef BME280_FLOAT_ENABLE
	printf("temperature:%0.2f*C   pressure:%0.2fhPa   humidity:%0.2f%%\r\n",comp_data->temperature, comp_data->pressure/100, comp_data->humidity);
#else
	printf("temperature:%ld*C   pressure:%ldhPa   humidity:%ld%%\r\n",comp_data->temperature, comp_data->pressure/100, comp_data->humidity);
#endif
}

int8_t stream_sensor_data_forced_mode(struct bme280_dev *dev)
{
    int8_t rslt;
    uint8_t settings_sel;
    struct bme280_data comp_data;

    /* Recommended mode of operation: Indoor navigation */
    dev->settings.osr_h = BME280_OVERSAMPLING_1X;
    dev->settings.osr_p = BME280_OVERSAMPLING_16X;
    dev->settings.osr_t = BME280_OVERSAMPLING_2X;
    dev->settings.filter = BME280_FILTER_COEFF_16;

    settings_sel = BME280_OSR_PRESS_SEL | BME280_OSR_TEMP_SEL | BME280_OSR_HUM_SEL | BME280_FILTER_SEL;

    rslt = bme280_set_sensor_settings(settings_sel, dev);

    printf("Temperature           Pressure             Humidity\r\n");
    /* Continuously stream sensor data */
    while (1) {
        rslt = bme280_set_sensor_mode(BME280_FORCED_MODE, dev);
        /* Wait for the measurement to complete and print data @25Hz */
        dev->delay_ms(40);
        rslt = bme280_get_sensor_data(BME280_ALL, &comp_data, dev);
        print_sensor_data(&comp_data);
    }
    return rslt;
}


int8_t stream_sensor_data_normal_mode(struct bme280_dev *dev)
{
	int8_t rslt;
	uint8_t settings_sel;
	struct bme280_data comp_data;

	/* Recommended mode of operation: Indoor navigation */
	dev->settings.osr_h = BME280_OVERSAMPLING_1X;
	dev->settings.osr_p = BME280_OVERSAMPLING_16X;
	dev->settings.osr_t = BME280_OVERSAMPLING_2X;
	dev->settings.filter = BME280_FILTER_COEFF_16;
	dev->settings.standby_time = BME280_STANDBY_TIME_62_5_MS;

	settings_sel = BME280_OSR_PRESS_SEL;
	settings_sel |= BME280_OSR_TEMP_SEL;
	settings_sel |= BME280_OSR_HUM_SEL;
	settings_sel |= BME280_STANDBY_SEL;
	settings_sel |= BME280_FILTER_SEL;
	rslt = bme280_set_sensor_settings(settings_sel, dev);
	rslt = bme280_set_sensor_mode(BME280_NORMAL_MODE, dev);

	printf("Temperature           Pressure             Humidity\r\n");
	while (1) {
		/* Delay while the sensor completes a measurement */
		dev->delay_ms(70);
		rslt = bme280_get_sensor_data(BME280_ALL, &comp_data, dev);
		print_sensor_data(&comp_data);
	}

	return rslt;
}

#if(USEIIC)
int main(int argc, char* argv[])
{
  struct bme280_dev dev;
  int8_t rslt = BME280_OK;

  if ((fd = open(IIC_Dev, O_RDWR)) < 0) {
    printf("Failed to open the i2c bus %s", argv[1]);
    exit(1);
  }
  if (ioctl(fd, I2C_SLAVE, 0x77) < 0) {
    printf("Failed to acquire bus access and/or talk to slave.\n");
    exit(1);
  }
  //dev.dev_id = BME280_I2C_ADDR_PRIM;//0x76
  dev.dev_id = BME280_I2C_ADDR_SEC; //0x77
  dev.intf = BME280_I2C_INTF;
  dev.read = user_i2c_read;
  dev.write = user_i2c_write;
  dev.delay_ms = user_delay_ms;

  rslt = bme280_init(&dev);
  printf("\r\n BME280 Init Result is:%d \r\n",rslt);
  //stream_sensor_data_forced_mode(&dev);
  stream_sensor_data_normal_mode(&dev);
}
#else
int main(int argc, char* argv[])
{
  // Open GPIO chip (usually "gpiochip0")
  chip = gpiod_chip_open_by_name("gpiochip0");
  if (!chip) {
    perror("gpiod_chip_open_by_name");
    return 1;
  }
  cs_line = gpiod_chip_get_line(chip, CS_GPIO);
  if (!cs_line) {
    perror("gpiod_chip_get_line");
    gpiod_chip_close(chip);
    return 1;
  }
  if (gpiod_line_request_output(cs_line, "bme280_cs", 1) < 0) {
    perror("gpiod_line_request_output");
    gpiod_chip_close(chip);
    return 1;
  }

  // Open SPI device
  spi_fd = open(SPI_DEV, O_RDWR);
  if (spi_fd < 0) {
    perror("SPI device open");
    gpiod_line_release(cs_line);
    gpiod_chip_close(chip);
    return 1;
  }
  uint8_t mode = 0;
  uint32_t speed = 2000000;
  ioctl(spi_fd, SPI_IOC_WR_MODE, &mode);
  ioctl(spi_fd, SPI_IOC_WR_MAX_SPEED_HZ, &speed);

  SPI_BME280_CS_High(); // Deselect by default

  struct bme280_dev dev;
  int8_t rslt = BME280_OK;

  dev.dev_id = 0;
  dev.intf = BME280_SPI_INTF;
  dev.read = user_spi_read;
  dev.write = user_spi_write;
  dev.delay_ms = user_delay_ms;

  rslt = bme280_init(&dev);
  printf("\r\n BME280 Init Result is:%d \r\n",rslt);
  //stream_sensor_data_forced_mode(&dev);
  stream_sensor_data_normal_mode(&dev);

  close(spi_fd);
  gpiod_line_release(cs_line);
  gpiod_chip_close(chip);
  return 0;
}
#endif
