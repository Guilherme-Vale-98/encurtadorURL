package routes

import (
	"log"
	"os"
	"strconv"
	"time"
	"github.com/Guilherme-Vale-98/encurtadorURL/database"
	"github.com/Guilherme-Vale-98/encurtadorURL/helpers"
	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)
type request struct{
	URL string `json:"url"`	
	CustomShort string `json:"short"`	
	Expiry time.Duration `json:"expiry"`	
}

type response struct{
	URL string `json:"url"`	
	CustomShort string `json:"short"`	
	Expiry time.Duration `json:"expiry"`	
	XRateRemaining int `json:"rate_limit"`	
	XRateLimitReset time.Duration `json:"rate_limit_reset"`	
}

func ShortenURL(c *fiber.Ctx) error {
	//1- Rate limit 2- validate URL 3- domain error 4- enforce https

	body := new(request)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Error": "Can not parse JSON",
		})
	}

	if !govalidator.IsURL(body.URL){
		log.Printf("Accessed invalid URL by %s", c.IP())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Error": "Invalid URL",
		})
	}

	if !helpers.RemoveDomainError(body.URL){
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"Error":"Invalid domain",
		})
	}



	r2 := database.CreateClient(1)

	defer r2.Close()

	val, err := r2.Get(database.Ctx, c.IP()).Result()

	if err ==  redis.Nil{
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else{
		val, _ = r2.Get(database.Ctx, c.IP()).Result()
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"rate_limit_reset": limit/time.Nanosecond/ time.Minute,
			})
		}

	}




	body.URL = helpers.EnforceHTTP(body.URL)

	var id string

	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()

	val, _ = r.Get(database.Ctx, id).Result()

	if val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "URL custom short is already use",
		})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3000*time.Second).Err()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect to server",
		})
	}

	resp := response{
		URL: body.URL,
		CustomShort: "",
		Expiry: body.Expiry,
		XRateRemaining: 10,
		XRateLimitReset: 30,
	}

	r2.Decr(database.Ctx, c.IP())


	val,_ = r2.Get(database.Ctx, c.IP()).Result()

	resp.XRateRemaining, _ = strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	return c.Status(fiber.StatusOK).JSON(resp)

}
